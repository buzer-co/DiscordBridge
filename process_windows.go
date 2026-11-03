//go:build windows

package main

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

// Only the bundled process is assigned. Closing the application closes its
// non-inheritable job handle and terminates that process without a PID file.
func attachOwnedJob(pid int) (uintptr, error) {
	create := kernel32.NewProc("CreateJobObjectW")
	set := kernel32.NewProc("SetInformationJobObject")
	assign := kernel32.NewProc("AssignProcessToJobObject")
	job, _, err := create.Call(0, 0)
	if job == 0 {
		return 0, fmt.Errorf("create owned process group: %w", err)
	}
	var limits [144]byte                                 // x64 JOBOBJECT_EXTENDED_LIMIT_INFORMATION
	binary.LittleEndian.PutUint32(limits[16:20], 0x2000) // KILL_ON_JOB_CLOSE
	ok, _, err := set.Call(job, 9, uintptr(unsafe.Pointer(&limits[0])), uintptr(len(limits)))
	if ok == 0 {
		call(procCloseHandle, job)
		return 0, fmt.Errorf("configure owned process group: %w", err)
	}
	h, err := syscall.OpenProcess(0x100|1, false, uint32(pid)) // SET_QUOTA | TERMINATE
	if err != nil {
		call(procCloseHandle, job)
		return 0, err
	}
	defer syscall.CloseHandle(h)
	ok, _, err = assign.Call(job, uintptr(h))
	if ok == 0 {
		call(procCloseHandle, job)
		return 0, fmt.Errorf("attach owned engine process: %w", err)
	}
	return job, nil
}

func (ap *app) purgeLegacyLogs() {
	// Exact app-owned names only; old settings and DNS restore backups survive.
	for _, name := range []string{"session.log", "goodbyedpi.log"} {
		if err := os.Remove(filepath.Join(ap.dataDir, name)); err != nil && !os.IsNotExist(err) {
			ap.detail = ap.L("Eski günlük dosyası silinemedi; yeni günlük tutulmuyor.", "An old log file could not be removed; new logging is disabled.")
		}
	}
}
