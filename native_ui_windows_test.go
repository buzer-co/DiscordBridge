//go:build windows

package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
	"unsafe"
)

var testGetUpdateRect = user32.NewProc("GetUpdateRect")
var testValidateRect = user32.NewProc("ValidateRect")
var testRedrawWindow = user32.NewProc("RedrawWindow")
var testGetDC = user32.NewProc("GetDC")
var testReleaseDC = user32.NewProc("ReleaseDC")
var testGetDIBits = gdi32.NewProc("GetDIBits")
var testGetGuiResources = user32.NewProc("GetGuiResources")
var testGetCurrentProcess = kernel32.NewProc("GetCurrentProcess")

func TestNativeWindowRepaintAndResources(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	a = &app{root: t.TempDir(), dataDir: t.TempDir(), uiState: uiState{status: "Bekliyor", mode: "auto", selectedProfile: "special", language: "tr"}, buttons: map[int]uintptr{}, styles: map[uint32]style{}, enabled: map[int]bool{}, done: make(chan struct{})}
	if err := a.createMainWindow(); err != nil {
		t.Fatal(err)
	}
	defer call(procDestroyWindow, a.hwnd)
	call(procShowWindow, a.hwnd, swShow)
	call(testRedrawWindow, a.hwnd, 0, 0, 0x181)
	if a.gdiToken == 0 || a.bufferDC == 0 {
		t.Fatal("native graphics initialization failed")
	}
	for i := 0; i < 20; i++ {
		a.enqueue(event{trafficSample: true, trafficRX: float64(i), trafficTX: float64(i) / 2})
		a.update()
		call(procUpdateWindow, a.hwnd)
	}
	call(testValidateRect, a.hwnd, 0)
	for _, h := range a.buttons {
		call(testValidateRect, h, 0)
	}
	beforeCaption := a.styles[idStart]
	for i := 0; i < 1000; i++ {
		a.enqueue(event{trafficSample: true, trafficRX: float64(i), trafficTX: float64(i) / 3})
	}
	a.update()
	var client rect
	call(procGetClientRect, a.hwnd, uintptr(unsafe.Pointer(&client)))
	if client.right != a.px(1080) || client.bottom != a.px(710) {
		t.Fatalf("client clipped: %v at scale %f", client, a.scale)
	}
	expected := a.scaledRect(networkRect)
	var dirty rect
	if call(testGetUpdateRect, a.hwnd, uintptr(unsafe.Pointer(&dirty)), 0) == 0 || dirty != expected {
		t.Fatalf("traffic invalidated %v; expected only %v", dirty, expected)
	}
	for id, h := range a.buttons {
		if call(testGetUpdateRect, h, uintptr(unsafe.Pointer(&dirty)), 0) != 0 {
			t.Fatalf("traffic invalidated button %d: %v", id, dirty)
		}
	}
	if a.styles[idStart] != beforeCaption {
		t.Fatal("traffic changed the main action")
	}
	call(procUpdateWindow, a.hwnd)
	process := call(testGetCurrentProcess)
	before := call(testGetGuiResources, process, 0)
	for i := 0; i < 250; i++ {
		a.enqueue(event{trafficSample: true, trafficRX: float64(i % 30), trafficTX: float64(i % 10)})
		a.update()
		call(procUpdateWindow, a.hwnd)
	}
	after := call(testGetGuiResources, process, 0)
	if before == 0 || after > before+8 {
		t.Fatalf("GDI resources grew across repaints: %d -> %d", before, after)
	}
	call(procSendMessage, a.hwnd, wmCommand, idProfileSpecial, 0)
	if a.mode != "manual" || a.selectedProfile != "special" {
		t.Fatal("native profile action failed")
	}
	a.lastWorkingProfile = "special"
	a.saveConfig()
	other := &app{dataDir: a.dataDir}
	other.loadConfig()
	if other.mode != "auto" || other.lastWorkingProfile != "special" {
		t.Fatal("profile not persisted")
	}
	call(procSendMessage, a.hwnd, wmCommand, idLanguage, 0)
	if a.language != "en" || a.styles[idOpenDiscord].caption != "Open Discord" {
		t.Fatal("language action failed")
	}
	call(procSendMessage, a.hwnd, wmCommand, idLanguage, 0)
	call(testRedrawWindow, a.hwnd, 0, 0, 0x181)
	saveNativeScreenshot(t, a)
	for _, id := range []int{idDNS, idRestore, idTest, idOpenLog, idPack} {
		call(procSendMessage, a.hwnd, wmCommand, uintptr(id), 0)
		if a.page != id || call(procIsWindowVisible, a.logEdit) == 0 || call(procIsWindowVisible, a.buttons[idStart]) != 0 {
			t.Fatalf("tool %d did not open its visible page", id)
		}
		if a.styles[idToolAction].caption == "" {
			t.Fatal("tool action has no label")
		}
		if id == idRestore && a.enabled[idToolAction] {
			t.Fatal("restore enabled without a backup")
		}
		call(testRedrawWindow, a.hwnd, 0, 0, 0x181)
		saveNativeScreenshot(t, a, fmt.Sprintf("windows-tool-%d.png", id))
	}
	// A test executable has no embedded payload: the actual Verify action must
	// finish visibly with an error rather than silently writing a hidden log.
	call(procSendMessage, a.hwnd, wmCommand, idToolAction, 0)
	deadline := time.Now().Add(3 * time.Second)
	for a.busy && time.Now().Before(deadline) {
		a.update()
		time.Sleep(10 * time.Millisecond)
	}
	a.update()
	if a.busy || a.toolResult(idPack).state != "error" || a.toolResult(idPack).message == "" {
		t.Fatal("tool failure was not surfaced")
	}
	call(testRedrawWindow, a.hwnd, 0, 0, 0x181)
	saveNativeScreenshot(t, a, "windows-tool-result.png")
	call(procSendMessage, a.hwnd, wmCommand, idOverview, 0)
	if a.page != 0 || call(procIsWindowVisible, a.buttons[idStart]) == 0 || call(procIsWindowVisible, a.logEdit) != 0 {
		t.Fatal("overview navigation failed")
	}
	if call(user32.NewProc("GetWindowTextLengthW"), a.logEdit) != 0 {
		t.Fatal("hidden result field retained text")
	}
	for _, id := range []int{idProfile7, idProfile8} {
		call(procSendMessage, a.hwnd, wmCommand, uintptr(id), 0)
		if a.selectedProfile != profileIDs[profileIndex(id)] {
			t.Fatal("new profile selection failed")
		}
	}
	a.mode = "auto"
	a.lastWorkingProfile = "alt7"
	a.saveConfig()
	reloaded := &app{dataDir: a.dataDir}
	reloaded.loadConfig()
	if reloaded.lastWorkingProfile != "alt7" || reloaded.profilePlan()[0].id != "alt7" {
		t.Fatal("last successful profile was not preferred")
	}
	raw, err := os.ReadFile(filepath.Join(a.dataDir, "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	var settings map[string]any
	if json.Unmarshal(raw, &settings) != nil || len(settings) != 1 {
		t.Fatal("settings contain more than profile preferences")
	}
	a.logLine("Transient result only")
	for _, name := range []string{"session.log", "goodbyedpi.log", "process.json"} {
		if _, err := os.Stat(filepath.Join(a.dataDir, name)); !os.IsNotExist(err) {
			t.Fatalf("unexpected persistent record %s", name)
		}
	}
	if !a.addTray() {
		t.Log("notification area unavailable on this CI desktop")
	} else {
		a.hideToTray()
		if call(procIsWindowVisible, a.hwnd) != 0 {
			t.Fatal("tray hide failed")
		}
		a.restoreFromTray()
		if call(procIsWindowVisible, a.hwnd) == 0 {
			t.Fatal("tray restore failed")
		}
	}
	t.Logf("Native Windows paint: 1000-event burst, 250 redraws; GDI objects %d -> %d; profiles/language/tray checked", before, after)
}

func saveNativeScreenshot(t *testing.T, ap *app, names ...string) {
	if os.Getenv("CI") == "" {
		return
	}
	var client rect
	call(procGetClientRect, ap.hwnd, uintptr(unsafe.Pointer(&client)))
	w, h := int(client.right), int(client.bottom)
	dc := call(testGetDC, ap.hwnd)
	if dc == 0 {
		t.Fatal("GetDC failed")
	}
	defer call(testReleaseDC, ap.hwnd, dc)
	mem := call(procCreateCompatibleDC, dc)
	bmp := call(procCreateCompatibleBitmap, dc, uintptr(w), uintptr(h))
	old := call(procSelectObject, mem, bmp)
	call(procBitBlt, mem, 0, 0, uintptr(w), uintptr(h), dc, 0, 0, 0x00CC0020)
	call(procSelectObject, mem, old)
	defer call(procDeleteObject, bmp)
	defer call(procDeleteDC, mem)
	info := struct {
		size                   uint32
		width, height          int32
		planes, bits           uint16
		compression, sizeImage uint32
		x, y                   int32
		used, important        uint32
	}{size: 40, width: int32(w), height: -int32(h), planes: 1, bits: 32}
	pixels := make([]byte, w*h*4)
	if call(testGetDIBits, dc, bmp, 0, uintptr(h), uintptr(unsafe.Pointer(&pixels[0])), uintptr(unsafe.Pointer(&info)), 0) != uintptr(h) {
		t.Fatal("GetDIBits failed")
	}
	im := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < len(pixels); i += 4 {
		im.Pix[i], im.Pix[i+1], im.Pix[i+2], im.Pix[i+3] = pixels[i+2], pixels[i+1], pixels[i], 255
	}
	if err := os.MkdirAll("test-results", 0755); err != nil {
		t.Fatal(err)
	}
	name := "windows-ui.png"
	if len(names) > 0 {
		name = names[0]
	}
	f, err := os.Create(filepath.Join("test-results", name))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err = png.Encode(f, im); err != nil {
		t.Fatal(err)
	}
}
