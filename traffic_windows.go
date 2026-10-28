//go:build windows

package main

import (
	"fmt"
	"syscall"
	"time"
	"unsafe"
)

// Native 64-bit counters: no PowerShell process is spawned for graph updates.
// Layout follows the Windows SDK MIB_IF_ROW2, including its natural x64 padding.
var ipHelper = syscall.NewLazyDLL("iphlpapi.dll")
var getIfTable2 = ipHelper.NewProc("GetIfTable2")
var freeMibTable = ipHelper.NewProc("FreeMibTable")

func readPhysicalCounters() (map[uint64]byteCounter, error) {
	var table uintptr
	if code := call(getIfTable2, uintptr(unsafe.Pointer(&table))); code != 0 {
		return nil, fmt.Errorf("GetIfTable2: %d", code)
	}
	defer call(freeMibTable, table)
	count := *(*uint32)(unsafe.Pointer(table))
	if count > 4096 {
		return nil, fmt.Errorf("invalid interface table")
	}
	result := make(map[uint64]byteCounter)
	for i := uint32(0); i < count; i++ {
		r := (*ifRow2)(unsafe.Pointer(table + 8 + uintptr(i)*unsafe.Sizeof(ifRow2{})))
		// HardwareInterface + Up; exclude filters, loopback and VPN adapters.
		if r.Flags&1 != 0 && r.Flags&2 == 0 && r.OperStatus == 1 && r.Type != 24 && r.Type != 131 {
			result[r.Luid] = byteCounter{r.InOctets, r.OutOctets}
		}
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("no physical interface online")
	}
	return result, nil
}

func (ap *app) monitorTraffic() {
	previous, _ := readPhysicalCounters()
	last := time.Now()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ap.done:
			return
		case <-ticker.C:
			next, err := readPhysicalCounters()
			now := time.Now()
			if err != nil {
				previous = nil
				ap.enqueue(event{trafficUnavailable: true})
			} else if len(previous) > 0 {
				rx, tx := counterDelta(previous, next)
				sec := now.Sub(last).Seconds()
				if sec > 0 {
					ap.enqueue(event{trafficSample: true, trafficRX: float64(rx) / sec / 1024, trafficTX: float64(tx) / sec / 1024})
				}
			}
			previous, last = next, now
		}
	}
}
