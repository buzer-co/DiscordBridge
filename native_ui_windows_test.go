//go:build windows

package main

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"testing"
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
	var dirty rect
	if call(testGetUpdateRect, a.hwnd, uintptr(unsafe.Pointer(&dirty)), 0) == 0 || dirty != networkRect {
		t.Fatalf("traffic invalidated %v; expected only %v", dirty, networkRect)
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
	other := &app{dataDir: a.dataDir}
	other.loadConfig()
	if other.mode != "manual" || other.selectedProfile != "special" {
		t.Fatal("profile not persisted")
	}
	call(procSendMessage, a.hwnd, wmCommand, idLanguage, 0)
	if a.language != "en" || a.styles[idOpenDiscord].caption != "Open Discord" {
		t.Fatal("language action failed")
	}
	call(procSendMessage, a.hwnd, wmCommand, idLanguage, 0)
	call(testRedrawWindow, a.hwnd, 0, 0, 0x181)
	saveNativeScreenshot(t, a)
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

func saveNativeScreenshot(t *testing.T, ap *app) {
	if os.Getenv("CI") == "" {
		return
	}
	const w, h = 1080, 710
	dc := call(testGetDC, ap.hwnd)
	if dc == 0 {
		t.Fatal("GetDC failed")
	}
	defer call(testReleaseDC, ap.hwnd, dc)
	mem := call(procCreateCompatibleDC, dc)
	bmp := call(procCreateCompatibleBitmap, dc, w, h)
	old := call(procSelectObject, mem, bmp)
	call(procBitBlt, mem, 0, 0, w, h, dc, 0, 0, 0x00CC0020)
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
	}{size: 40, width: w, height: -h, planes: 1, bits: 32}
	pixels := make([]byte, w*h*4)
	if call(testGetDIBits, dc, bmp, 0, h, uintptr(unsafe.Pointer(&pixels[0])), uintptr(unsafe.Pointer(&info)), 0) != h {
		t.Fatal("GetDIBits failed")
	}
	im := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < len(pixels); i += 4 {
		im.Pix[i], im.Pix[i+1], im.Pix[i+2], im.Pix[i+3] = pixels[i+2], pixels[i+1], pixels[i], 255
	}
	if err := os.MkdirAll("test-results", 0755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(filepath.Join("test-results", "windows-ui.png"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err = png.Encode(f, im); err != nil {
		t.Fatal(err)
	}
}
