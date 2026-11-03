//go:build windows

package main

import (
	"os/exec"
	"syscall"
	"testing"
	"time"
)

func TestOwnedProcessGroupStopsWithoutPersistentPIDFile(t *testing.T) {
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", "Start-Sleep -Seconds 30")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer cmd.Process.Kill()
	job, err := attachOwnedJob(cmd.Process.Pid)
	if err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	call(procCloseHandle, job)
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("owned process survived closing its job")
	}
}
