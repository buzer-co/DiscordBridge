//go:build windows

package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
	"unsafe"
)

func dnsSettingsSnapshot(t *testing.T) string {
	t.Helper()
	s, err := powershell("$ErrorActionPreference='Stop'; @(Get-DnsClientServerAddress | Sort-Object InterfaceIndex,AddressFamily | Select-Object InterfaceIndex,AddressFamily,ServerAddresses) | ConvertTo-Json -Depth 5 -Compress")
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestNativeSelectiveDNS(t *testing.T) {
	if call(procIsAdmin) == 0 {
		if os.Getenv("CI") != "" {
			t.Fatal("native DNS driver test requires administrator rights")
		}
		t.Skip("administrator rights required for native packet interception")
	}
	if unsafe.Sizeof(divertAddress{}) != 80 {
		t.Fatal("WinDivert ABI size mismatch")
	}
	before := dnsSettingsSnapshot(t)
	// Windows may retain the loaded .sys file until reboot even after every
	// WinDivert handle is closed. Do not stop a shared global driver service.
	dir, err := os.MkdirTemp("", "discordbridge-native-dns-")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.RemoveAll(dir); err != nil {
			var pathErr *os.PathError
			if errors.As(err, &pathErr) && filepath.Base(pathErr.Path) == "WinDivert64.sys" && errors.Is(err, syscall.ERROR_ACCESS_DENIED) {
				t.Log("Windows retains the verified WinDivert driver until reboot; global service left untouched")
			} else {
				t.Error(err)
			}
		}
	}()
	payload, err := os.ReadFile("engine.zip")
	if err != nil {
		t.Fatal(err)
	}
	if err = extractPayload(payload, dir); err != nil {
		t.Fatal(err)
	}
	var calls atomic.Int32
	bridge, err := startDNSBridge(dir, func(ctx context.Context, q []byte) ([]byte, error) { calls.Add(1); return testDNSAnswer(q), nil }, nil, func(err error) { t.Log(err) })
	if err != nil {
		t.Fatal(err)
	}
	defer bridge.Close()
	query := func(name string) []byte {
		t.Helper()
		c, err := net.DialTimeout("udp4", "192.0.2.53:53", time.Second)
		if err != nil {
			t.Fatal(err)
		}
		defer c.Close()
		c.SetDeadline(time.Now().Add(4 * time.Second))
		q := testDNSQuery(name)
		if _, err = c.Write(q); err != nil {
			t.Fatal(err)
		}
		buf := make([]byte, 2048)
		n, err := c.Read(buf)
		if err != nil {
			t.Fatal(err)
		}
		if validateDNSResponse(q, buf[:n]) != nil {
			t.Fatal("invalid native DNS response")
		}
		return buf[:n]
	}
	// No service exists at this TEST-NET address. A response proves that the
	// native outbound interception, reply direction, addresses and checksums work.
	if !bytes.Equal(query("updates.discord.com"), testDNSAnswer(testDNSQuery("updates.discord.com"))) {
		t.Fatal("wrong target reply")
	}
	if calls.Load() != 1 {
		t.Fatal("DoH exchange count", calls.Load())
	}

	// A second, lower-priority handle acts as a deterministic mock ISP DNS.
	// It sees only packets passed through by the application's filter and drops
	// them locally, so no test DNS packets leave for the TEST-NET destination.
	open, err := bridge.dll.FindProc("WinDivertOpen")
	if err != nil {
		t.Fatal(err)
	}
	filter, _ := syscall.BytePtrFromString("outbound and udp and udp.DstPort == 53 and ip.DstAddr == 192.0.2.53")
	low, _, openErr := open.Call(uintptr(unsafe.Pointer(filter)), 0, 9, 0)
	if low == ^uintptr(0) {
		t.Fatal(openErr)
	}
	done := make(chan error, 1)
	go func() {
		buf := make([]byte, 2048)
		var n uint32
		var addr divertAddress
		ok, _, err := bridge.recv.Call(low, uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)), uintptr(unsafe.Pointer(&n)), uintptr(unsafe.Pointer(&addr)))
		if ok == 0 {
			done <- err
			return
		}
		p, _ := discordDNSPacket(buf[:n])
		if !bytes.Equal(p.query, testDNSQuery("example.com")) {
			done <- fmt.Errorf("unrelated query payload changed")
			return
		}
		addr.Flags &= 1 << 20
		done <- bridge.inject(p.response(testDNSAnswer(p.query)), &addr)
	}()
	defer func() {
		if low != 0 {
			bridge.shutdown.Call(low, 1)
			bridge.closeHandle.Call(low)
		}
	}()
	query("example.com")
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(4 * time.Second):
		t.Fatal("pass-through capture timed out")
	}
	if calls.Load() != 1 {
		t.Fatal("unrelated DNS was sent to DoH")
	}
	bridge.shutdown.Call(low, 1)
	bridge.closeHandle.Call(low)
	low = 0
	bridge.Close()
	after := dnsSettingsSnapshot(t)
	if strings.TrimSpace(before) != strings.TrimSpace(after) {
		t.Fatal("network adapter DNS settings changed")
	}
	t.Log("Native WinDivert DNS reply, unrelated-query pass-through, clean stop and unchanged adapter DNS verified")
}
