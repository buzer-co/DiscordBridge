//go:build windows

package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

// WinDivert 2.2 WINDIVERT_ADDRESS, from the bundled unmodified LGPL source.
type divertAddress struct {
	Timestamp int64
	Flags     uint32
	Reserved  uint32
	Data      [64]byte
}

type dnsBridge struct {
	handle                                      uintptr
	dll                                         *syscall.DLL
	recv, send, checksum, shutdown, closeHandle *syscall.Proc
	cancel                                      context.CancelFunc
	done                                        chan struct{}
	once                                        sync.Once
	cleanup                                     func()
}

func startDNSBridge(engineDir string, exchange dnsExchange, cleanup func(), onError func(error)) (*dnsBridge, error) {
	// Load only the previously verified, bundled library via an absolute path.
	if err := checkEngineFiles(filepath.Join(engineDir, "goodbyedpi.exe")); err != nil {
		return nil, err
	}
	path, err := filepath.Abs(filepath.Join(engineDir, "WinDivert.dll"))
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil || fmt.Sprintf("%x", sha256.Sum256(data)) != "6110bfa44667405179c3e15e12af1b62037e447ed59b054b19042032995e6c7e" {
		return nil, fmt.Errorf("bundled WinDivert library hash verification failed")
	}
	dll, err := syscall.LoadDLL(path)
	if err != nil {
		return nil, err
	}
	b := &dnsBridge{dll: dll, done: make(chan struct{}), cleanup: cleanup}
	procs := []struct {
		name string
		dest **syscall.Proc
	}{
		{"WinDivertRecv", &b.recv}, {"WinDivertSend", &b.send},
		{"WinDivertHelperCalcChecksums", &b.checksum}, {"WinDivertShutdown", &b.shutdown}, {"WinDivertClose", &b.closeHandle},
	}
	for _, p := range procs {
		*p.dest, err = dll.FindProc(p.name)
		if err != nil {
			dll.Release()
			return nil, err
		}
	}
	open, err := dll.FindProc("WinDivertOpen")
	if err != nil {
		dll.Release()
		return nil, err
	}
	filter, _ := syscall.BytePtrFromString("outbound and !loopback and !impostor and udp and udp.DstPort == 53")
	h, _, openErr := open.Call(uintptr(unsafe.Pointer(filter)), 0, 10, 0)
	if h == ^uintptr(0) {
		dll.Release()
		return nil, fmt.Errorf("Discord DNS filter: %w", openErr)
	}
	b.handle = h
	ctx, cancel := context.WithCancel(context.Background())
	b.cancel = cancel
	go b.run(ctx, exchange, onError)
	return b, nil
}

func (b *dnsBridge) inject(wire []byte, addr *divertAddress) error {
	if len(wire) == 0 {
		return nil
	}
	ok, _, err := b.checksum.Call(uintptr(unsafe.Pointer(&wire[0])), uintptr(len(wire)), uintptr(unsafe.Pointer(addr)), 0)
	if ok == 0 {
		return fmt.Errorf("DNS packet checksum: %w", err)
	}
	var n uint32
	ok, _, err = b.send.Call(b.handle, uintptr(unsafe.Pointer(&wire[0])), uintptr(len(wire)), uintptr(unsafe.Pointer(&n)), uintptr(unsafe.Pointer(addr)))
	if ok == 0 || int(n) != len(wire) {
		return fmt.Errorf("DNS packet delivery: %w", err)
	}
	return nil
}

func (b *dnsBridge) run(ctx context.Context, exchange dnsExchange, onError func(error)) {
	var work sync.WaitGroup
	slots := make(chan struct{}, 32)
	defer func() { work.Wait(); close(b.done) }()
	buffer := make([]byte, 65575)
	report := func(err error) {
		if err != nil && ctx.Err() == nil && onError != nil {
			onError(err)
		}
	}
	for {
		var n uint32
		var addr divertAddress
		ok, _, err := b.recv.Call(b.handle, uintptr(unsafe.Pointer(&buffer[0])), uintptr(len(buffer)), uintptr(unsafe.Pointer(&n)), uintptr(unsafe.Pointer(&addr)))
		if ok == 0 {
			report(fmt.Errorf("DNS filter stopped: %w", err))
			return
		}
		wire := buffer[:n]
		_, target := discordDNSPacket(wire)
		if !target || ctx.Err() != nil {
			report(b.inject(wire, &addr))
			continue
		}
		// Copy before the next receive. Other domains never wait for HTTPS.
		packet, _ := discordDNSPacket(append([]byte(nil), wire...))
		replyAddr := addr
		replyAddr.Flags &= 1 << 20 // inbound IPv4/IPv6, network packet event.
		select {
		case slots <- struct{}{}:
			work.Add(1)
			go func(p dnsPacket, a divertAddress) {
				defer work.Done()
				defer func() { <-slots }()
				deadline, cancel := context.WithTimeout(ctx, 6500*time.Millisecond)
				defer cancel()
				response, err := exchange(deadline, p.query)
				if err == nil {
					err = validateDNSResponse(p.query, response)
				}
				if err != nil {
					response = dnsErrorResponse(p.query, 2, false)
				}
				// Never leak a selected query back to a poisoned plain DNS server.
				report(b.inject(p.response(response), &a))
			}(packet, replyAddr)
		default:
			report(b.inject(packet.response(dnsErrorResponse(packet.query, 2, false)), &replyAddr))
		}
	}
}

func (b *dnsBridge) Close() {
	if b == nil {
		return
	}
	b.once.Do(func() {
		b.cancel()
		// Stop receiving and drain captured packets before closing the handle.
		b.shutdown.Call(b.handle, 1) // WINDIVERT_SHUTDOWN_RECV
		<-b.done
		b.closeHandle.Call(b.handle)
		if b.cleanup != nil {
			b.cleanup()
		}
		b.dll.Release()
	})
}

func (ap *app) startDiscordDNS() error {
	ap.dnsMu.Lock()
	defer ap.dnsMu.Unlock()
	if ap.dnsBridge != nil {
		return nil
	}
	exchange, cleanup := newDiscordDoH()
	b, err := startDNSBridge(ap.dependencyRoot(), exchange, cleanup, func(err error) {
		ap.enqueue(event{line: ap.L("Discord DNS uyarısı: ", "Discord DNS warning: ") + err.Error()})
	})
	if err != nil {
		cleanup()
		return err
	}
	ap.dnsBridge = b
	// Discard previously poisoned answers, without changing any DNS configuration.
	if _, err := powershell("Clear-DnsClientCache -ErrorAction Stop"); err != nil {
		b.Close()
		ap.dnsBridge = nil
		return fmt.Errorf("DNS cache could not be cleared: %w", err)
	}
	ap.enqueue(event{dns: "Discord DoH", line: ap.L("Discord için şifreli DNS açık. Windows ve modem DNS ayarları değiştirilmedi.", "Discord encrypted DNS active. Windows and router DNS settings were not changed.")})
	return nil
}

func (ap *app) stopDiscordDNS() {
	ap.dnsMu.Lock()
	defer ap.dnsMu.Unlock()
	if ap.dnsBridge != nil {
		ap.dnsBridge.Close()
		ap.dnsBridge = nil
	}
}
