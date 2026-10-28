//go:build windows

// Discord Bridge v4.6: softer Discord-style Win32 control center, explicit Discord launch, persistent tray.
// Does not modify ExitLag. No telemetry or injected scripts.
package main

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"embed"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode/utf16"
	"unsafe"
)

const (
	versionText        = "4.6.0"
	payloadMagic       = "DBRIDGE3"
	wsOverlappedWindow = 0x00CF0000
	wsVisible          = 0x10000000
	wsChild            = 0x40000000
	wsTabstop          = 0x00010000
	wsVScroll          = 0x00200000
	wsBorder           = 0x00800000
	esReadOnly         = 0x0800
	esMultiline        = 0x0004
	esAutoVScroll      = 0x0040
	esAutoHScroll      = 0x0080
	bsOwnerDraw        = 0x000B
	wmDestroy          = 0x0002
	wmClose            = 0x0010
	wmTray             = 0x802A // WM_APP + 42; notification-area callback
	wmTrayRestore      = 0x802B // restore UI on explicit request
	wmLButtonDblClk    = 0x0203
	wmRButtonUp        = 0x0205
	wmContextMenu      = 0x007B
	swHide             = 0
	swRestore          = 9
	nimAdd             = 0x00000000
	nimDelete          = 0x00000002
	nifMessage         = 0x00000001
	nifIcon            = 0x00000002
	nifTip             = 0x00000004
	tpmRightButton     = 0x0002
	tpmReturnCmd       = 0x0100
	mfString           = 0x0000
	mfSeparator        = 0x0800
	trayIconID         = 91
	trayMenuShow       = 3001
	trayMenuDiscord    = 3002
	trayMenuExit       = 3003
	trayMenuStopExit   = 3004
	wmPaint            = 0x000F
	wmEraseBkgnd       = 0x0014
	wmSetCursor        = 0x0020
	wmCommand          = 0x0111
	wmDrawItem         = 0x002B
	wmCtlColorEdit     = 0x0133
	wmCtlColorStatic   = 0x0138
	wmAppEvent         = 0x8001
	wmSetFont          = 0x0030
	emSetSel           = 0x00B1
	emReplaceSel       = 0x00C2
	swShow             = 5
	swNormal           = 1
	colorWhite         = 0xFFFFFF
	gdiTransparent     = 1
	dtSingleLine       = 0x0020
	dtVCenter          = 0x0004
	dtCenter           = 0x0001
	dtNoPrefix         = 0x0800
	ofnFileMustExist   = 0x1000
	ofnPathMustExist   = 0x0800
	ofnExplorer        = 0x80000
	diSelected         = 0x0001
	idBrowse           = 1001
	idStart            = 1002
	idStop             = 1003
	idDNS              = 1004
	idRestore          = 1005
	idTest             = 1006
	idOpenDiscord      = 1007
	idProfile          = 1008
	idPack             = 1009
	idOpenLog          = 1010
	idLanguage         = 1011
	idProfileAuto      = 1050
	idProfileSpecial   = 1051
	idProfile5         = 1052
	idProfile6         = 1053
	idProfile9         = 1054
	idProfile2         = 1055
	mbYesNo            = 0x00000004
	mbIconQuestion     = 0x00000020
	mbIconWarning      = 0x00000030
	idYes              = 6
)

var (
	user32                        = syscall.NewLazyDLL("user32.dll")
	gdi32                         = syscall.NewLazyDLL("gdi32.dll")
	shell32                       = syscall.NewLazyDLL("shell32.dll")
	dwmapi                        = syscall.NewLazyDLL("dwmapi.dll")
	comdlg32                      = syscall.NewLazyDLL("comdlg32.dll")
	advapi32                      = syscall.NewLazyDLL("shell32.dll")
	kernel32                      = syscall.NewLazyDLL("kernel32.dll")
	procRegisterClass             = user32.NewProc("RegisterClassExW")
	procCreateWindow              = user32.NewProc("CreateWindowExW")
	procDefWindowProc             = user32.NewProc("DefWindowProcW")
	procShowWindow                = user32.NewProc("ShowWindow")
	procUpdateWindow              = user32.NewProc("UpdateWindow")
	procGetMessage                = user32.NewProc("GetMessageW")
	procTranslateMessage          = user32.NewProc("TranslateMessage")
	procDispatchMessage           = user32.NewProc("DispatchMessageW")
	procPostQuitMessage           = user32.NewProc("PostQuitMessage")
	procPostMessage               = user32.NewProc("PostMessageW")
	procSendMessage               = user32.NewProc("SendMessageW")
	procSetWindowText             = user32.NewProc("SetWindowTextW")
	procMessageBox                = user32.NewProc("MessageBoxW")
	procEnableWindow              = user32.NewProc("EnableWindow")
	procInvalidateRect            = user32.NewProc("InvalidateRect")
	procBeginPaint                = user32.NewProc("BeginPaint")
	procEndPaint                  = user32.NewProc("EndPaint")
	procGetClientRect             = user32.NewProc("GetClientRect")
	procFillRect                  = user32.NewProc("FillRect")
	procSetBkMode                 = gdi32.NewProc("SetBkMode")
	procSetBkColor                = gdi32.NewProc("SetBkColor")
	procSetTextColor              = gdi32.NewProc("SetTextColor")
	procSelectObject              = gdi32.NewProc("SelectObject")
	procDeleteObject              = gdi32.NewProc("DeleteObject")
	procCreateSolidBrush          = gdi32.NewProc("CreateSolidBrush")
	procCreatePen                 = gdi32.NewProc("CreatePen")
	procRoundRect                 = gdi32.NewProc("RoundRect")
	procEllipse                   = gdi32.NewProc("Ellipse")
	procDrawText                  = user32.NewProc("DrawTextW")
	procCreateFont                = gdi32.NewProc("CreateFontW")
	procGetOpenFileName           = comdlg32.NewProc("GetOpenFileNameW")
	procShellExecute              = shell32.NewProc("ShellExecuteW")
	procIsAdmin                   = shell32.NewProc("IsUserAnAdmin")
	procGetModule                 = kernel32.NewProc("GetModuleHandleW")
	procLoadIcon                  = user32.NewProc("LoadIconW")
	procDrawIconEx                = user32.NewProc("DrawIconEx")
	procDestroyWindow             = user32.NewProc("DestroyWindow")
	procSetForeground             = user32.NewProc("SetForegroundWindow")
	procCreatePopupMenu           = user32.NewProc("CreatePopupMenu")
	procAppendMenu                = user32.NewProc("AppendMenuW")
	procDestroyMenu               = user32.NewProc("DestroyMenu")
	procTrackPopupMenu            = user32.NewProc("TrackPopupMenu")
	procGetCursorPos              = user32.NewProc("GetCursorPos")
	procShellNotifyIcon           = shell32.NewProc("Shell_NotifyIconW")
	procFindWindow                = user32.NewProc("FindWindowW")
	procCreateMutex               = kernel32.NewProc("CreateMutexW")
	procGetLastError              = kernel32.NewProc("GetLastError")
	procCloseHandle               = kernel32.NewProc("CloseHandle")
	procDwmSetWindowAttribute     = dwmapi.NewProc("DwmSetWindowAttribute")
	procMoveToEx                  = gdi32.NewProc("MoveToEx")
	procLineTo                    = gdi32.NewProc("LineTo")
	procCreateCompatibleDC        = gdi32.NewProc("CreateCompatibleDC")
	procCreateCompatibleBitmap    = gdi32.NewProc("CreateCompatibleBitmap")
	procDeleteDC                  = gdi32.NewProc("DeleteDC")
	procStretchBlt                = gdi32.NewProc("StretchBlt")
	procSetStretchBltMode         = gdi32.NewProc("SetStretchBltMode")
	procSetBrushOrg               = gdi32.NewProc("SetBrushOrgEx")
	procSystemParametersInfo      = user32.NewProc("SystemParametersInfoW")
	procIsWindowVisible           = user32.NewProc("IsWindowVisible")
	procAdjustWindowRect          = user32.NewProc("AdjustWindowRectEx")
	procIsDialogMessage           = user32.NewProc("IsDialogMessageW")
	procSetCursor                 = user32.NewProc("SetCursor")
	procLoadCursor                = user32.NewProc("LoadCursorW")
	procDrawFocusRect             = user32.NewProc("DrawFocusRect")
	procGdiplusShutdown           = gdiplus.NewProc("GdiplusShutdown")
	procBitBlt                    = gdi32.NewProc("BitBlt")
	gdiplus                       = syscall.NewLazyDLL("gdiplus.dll")
	procGdiplusStartup            = gdiplus.NewProc("GdiplusStartup")
	procGdipCreateGraphicsFromHDC = gdiplus.NewProc("GdipCreateFromHDC")
	procGdipDeleteGraphics        = gdiplus.NewProc("GdipDeleteGraphics")
	procGdipSetSmoothingMode      = gdiplus.NewProc("GdipSetSmoothingMode")
	procGdipCreatePath            = gdiplus.NewProc("GdipCreatePath")
	procGdipAddPathLineI          = gdiplus.NewProc("GdipAddPathLineI")
	procGdipAddPathBezierI        = gdiplus.NewProc("GdipAddPathBezierI")
	procGdipClosePathFigure       = gdiplus.NewProc("GdipClosePathFigure")
	procGdipDeletePath            = gdiplus.NewProc("GdipDeletePath")
	procGdipCreateSolidFill       = gdiplus.NewProc("GdipCreateSolidFill")
	procGdipDeleteBrush           = gdiplus.NewProc("GdipDeleteBrush")
	procGdipFillPath              = gdiplus.NewProc("GdipFillPath")
	procGdipFillEllipseI          = gdiplus.NewProc("GdipFillEllipseI")
	procGdipCreatePen1            = gdiplus.NewProc("GdipCreatePen1")
	procGdipDrawPath              = gdiplus.NewProc("GdipDrawPath")
	procGdipDeletePen             = gdiplus.NewProc("GdipDeletePen")
)

type rect struct{ left, top, right, bottom int32 }
type point struct{ x, y int32 }
type wndclass struct {
	cbSize     uint32
	style      uint32
	wndProc    uintptr
	clsExtra   int32
	wndExtra   int32
	instance   uintptr
	icon       uintptr
	cursor     uintptr
	background uintptr
	menuName   *uint16
	className  *uint16
	iconSmall  uintptr
}
type message struct {
	hwnd    uintptr
	msg     uint32
	wparam  uintptr
	lparam  uintptr
	time    uint32
	pt      point
	private uint32
}
type paintStruct struct {
	hdc       uintptr
	erase     int32
	paint     rect
	restore   int32
	incUpdate int32
	reserved  [32]byte
}
type drawItem struct {
	ctlType    uint32
	ctlID      uint32
	itemID     uint32
	itemAction uint32
	itemState  uint32
	hwndItem   uintptr
	hdc        uintptr
	rc         rect
	itemData   uintptr
}

// NOTIFYICONDATAW layout on x64; reserved fields are kept for shell compatibility.
type notifyIconData struct {
	Size            uint32
	Hwnd            uintptr
	ID              uint32
	Flags           uint32
	CallbackMessage uint32
	Icon            uintptr
	Tip             [128]uint16
	State           uint32
	StateMask       uint32
	Info            [256]uint16
	Version         uint32
	InfoTitle       [64]uint16
	InfoFlags       uint32
	GUID            [16]byte
	BalloonIcon     uintptr
}

type openFileName struct {
	size            uint32
	hwnd            uintptr
	instance        uintptr
	filter          *uint16
	customFilter    *uint16
	maxCustomFilter uint32
	filterIndex     uint32
	file            *uint16
	maxFile         uint32
	fileTitle       *uint16
	maxFileTitle    uint32
	initialDir      *uint16
	title           *uint16
	flags           uint32
	offset          uint16
	extension       uint16
	defExt          *uint16
	custData        uintptr
	hook            uintptr
	template        *uint16
	reserved        uintptr
	reservedDW      uint32
	flagsEx         uint32
}

type style struct {
	caption string
	fill    uint32
	fg      uint32
}
type config struct {
	ExePath            string `json:"exe_path"`
	Mode               string `json:"mode"`
	SelectedProfile    string `json:"selected_profile"`
	LastWorkingProfile string `json:"last_working_profile"`
	Language           string `json:"language"`
}
type processInfo struct {
	PID     int    `json:"pid"`
	Path    string `json:"path"`
	Profile string `json:"profile,omitempty"`
}
type dnsBackup struct {
	Index       int      `json:"Index"`
	Alias       string   `json:"Alias"`
	GUID        string   `json:"GUID"`
	V4Manual    bool     `json:"V4Manual"`
	V6Manual    bool     `json:"V6Manual"`
	V4          []string `json:"V4"`
	V6          []string `json:"V6"`
	IPv6Enabled bool     `json:"IPv6Enabled"`
}

type app struct {
	scale        float64
	controlRects map[int]rect
	uiState
	hwnd, pathEdit, logEdit                             uintptr
	buttons                                             map[int]uintptr
	styles                                              map[uint32]style
	brushBG, brushCard, brushEdit, brushSidebar         uintptr
	gdiToken                                            uintptr
	hoverID                                             int
	done                                                chan struct{}
	bufferDC, bufferBitmap, bufferOld                   uintptr
	bufferW, bufferH                                    int32
	enabled                                             map[int]bool
	fTitle, fHeading, fText, fSmall, fBtn, fHero, fTiny uintptr
	root, dataDir, path                                 string
	queue                                               []event
	trayIcon                                            uintptr
	trayAdded                                           bool
	trayHintShown                                       bool
	mu                                                  sync.Mutex
}

var a *app

func ptr(s string) *uint16        { return syscall.StringToUTF16Ptr(s) }
func color(r, g, b uint32) uint32 { return r | (g << 8) | (b << 16) }

// Restful Discord-inspired neutrals and brand blurple. Every surface is softened.
var (
	bg           = color(49, 51, 56)
	card         = color(43, 45, 49)
	inputBG      = color(31, 32, 36)
	primary      = color(88, 101, 242)
	primarySoft  = color(64, 69, 118)
	teal         = color(67, 205, 159)
	red          = color(221, 103, 118)
	amber        = color(241, 185, 102)
	grey         = color(189, 192, 205)
	muted        = color(145, 149, 163)
	white        = color(247, 248, 252)
	sidebar      = color(34, 35, 40)
	surface      = color(56, 58, 65)
	edge         = color(75, 78, 90)
	hero         = color(51, 54, 79)
	hoverSurface = color(68, 70, 83)
	selectedBG   = color(66, 71, 116)
)

// Stable profile identifiers are persisted; titles are translated at display time.
type dpiProfile struct {
	id   string
	args []string
}

var dpiProfiles = []dpiProfile{
	{"special", []string{"-5", "--native-frag"}},
	{"standard5", []string{"-5"}},
	{"alt6", []string{"-6"}},
	{"alt9", []string{"-9"}},
	{"alt2", []string{"-2"}},
}
var profileButtonIDs = []int{idProfileAuto, idProfileSpecial, idProfile5, idProfile6, idProfile9, idProfile2}
var profileIDs = []string{"auto", "special", "standard5", "alt6", "alt9", "alt2"}

func profileValid(id string) bool {
	for _, p := range dpiProfiles {
		if p.id == id {
			return true
		}
	}
	return false
}
func (ap *app) L(tr, en string) string {
	if ap.language == "en" {
		return en
	}
	return tr
}
func (ap *app) profileName(id string) string {
	switch id {
	case "auto":
		return ap.L("Otomatik seçim", "Auto discovery")
	case "special":
		return "Superonline + ExitLag"
	case "standard5":
		return "Standart · -5"
	case "alt6":
		return "Alternatif · -6"
	case "alt9":
		return "Alternatif · -9"
	case "alt2":
		return "Alternatif · -2"
	case "direct":
		return ap.L("Doğrudan bağlantı", "Direct connection")
	}
	return id
}
func (ap *app) profilePlan() []dpiProfile {
	if ap.mode == "manual" {
		for _, p := range dpiProfiles {
			if p.id == ap.selectedProfile {
				return []dpiProfile{p}
			}
		}
		return []dpiProfile{dpiProfiles[0]}
	}
	result := make([]dpiProfile, 0, len(dpiProfiles))
	seen := map[string]bool{}
	// The very first discovery starts with the known working Superonline profile.
	// Once a working profile is learned, prefer it at later launches.
	if ap.lastWorkingProfile != "" {
		for _, p := range dpiProfiles {
			if p.id == ap.lastWorkingProfile {
				result = append(result, p)
				seen[p.id] = true
				break
			}
		}
	}
	for _, p := range dpiProfiles {
		if !seen[p.id] {
			result = append(result, p)
		}
	}
	return result
}

func call(p *syscall.LazyProc, values ...uintptr) uintptr { r, _, _ := p.Call(values...); return r }
func createFont(height, weight int32) uintptr {
	return call(procCreateFont, uintptr(int64(height)), 0, 0, 0, uintptr(weight), 0, 0, 0, 1, 0, 0, 5, 0, uintptr(unsafe.Pointer(ptr("Segoe UI"))))
}
func drawText(hdc uintptr, value string, x, y, w, h int32, clr uint32, font uintptr, flags uint32) {
	call(procSelectObject, hdc, font)
	call(procSetBkMode, hdc, gdiTransparent)
	call(procSetTextColor, hdc, uintptr(clr))
	r := rect{x, y, x + w, y + h}
	call(procDrawText, hdc, uintptr(unsafe.Pointer(ptr(value))), uintptr(^uint32(0)), uintptr(unsafe.Pointer(&r)), uintptr(flags|dtNoPrefix))
}

// GDI+ produces anti-aliased corners and translucent highlights instead of GDI rectangles.
// The app remains native Win32: no browser or external UI runtime required.
type gdipStartupInput struct {
	version                    uint32
	callback                   uintptr
	suppressBG, suppressCodecs uint32
}

func argb(c uint32, alpha uint8) uintptr {
	return uintptr(uint32(alpha)<<24 | (c&0x0000ff)<<16 | (c & 0x00ff00) | (c>>16)&0xff)
}
func (ap *app) initGraphics() {
	var token uintptr
	input := gdipStartupInput{version: 1}
	if call(procGdiplusStartup, uintptr(unsafe.Pointer(&token)), uintptr(unsafe.Pointer(&input)), 0) == 0 {
		ap.gdiToken = token
	}
}
func roundPath(r rect, radius int32) uintptr {
	var path uintptr
	if call(procGdipCreatePath, 0, uintptr(unsafe.Pointer(&path))) != 0 {
		return 0
	}
	x, y := r.left, r.top
	w, h := r.right-r.left, r.bottom-r.top
	if radius < 1 {
		radius = 1
	}
	if radius > w/2 {
		radius = w / 2
	}
	if radius > h/2 {
		radius = h / 2
	}
	R := radius
	K := int32(float64(R) * 0.55228475)
	line := func(x1, y1, x2, y2 int32) {
		call(procGdipAddPathLineI, path, uintptr(x1), uintptr(y1), uintptr(x2), uintptr(y2))
	}
	curve := func(x1, y1, x2, y2, x3, y3, x4, y4 int32) {
		call(procGdipAddPathBezierI, path, uintptr(x1), uintptr(y1), uintptr(x2), uintptr(y2), uintptr(x3), uintptr(y3), uintptr(x4), uintptr(y4))
	}
	line(x+R, y, x+w-R, y)
	curve(x+w-R, y, x+w-R+K, y, x+w, y+R-K, x+w, y+R)
	line(x+w, y+R, x+w, y+h-R)
	curve(x+w, y+h-R, x+w, y+h-R+K, x+w-R+K, y+h, x+w-R, y+h)
	line(x+w-R, y+h, x+R, y+h)
	curve(x+R, y+h, x+R-K, y+h, x, y+h-R+K, x, y+h-R)
	line(x, y+h-R, x, y+R)
	curve(x, y+R, x, y+R-K, x+R-K, y, x+R, y)
	call(procGdipClosePathFigure, path)
	return path
}
func paintRounded(hdc uintptr, r rect, clr uint32, radius int32, alpha uint8, stroke bool) {
	if a == nil || a.gdiToken == 0 {
		b := call(procCreateSolidBrush, uintptr(clr))
		p := call(procCreatePen, 0, 1, uintptr(clr))
		ob := call(procSelectObject, hdc, b)
		op := call(procSelectObject, hdc, p)
		call(procRoundRect, hdc, uintptr(r.left), uintptr(r.top), uintptr(r.right), uintptr(r.bottom), uintptr(radius*2), uintptr(radius*2))
		call(procSelectObject, hdc, ob)
		call(procSelectObject, hdc, op)
		call(procDeleteObject, p)
		call(procDeleteObject, b)
		return
	}
	var g uintptr
	if call(procGdipCreateGraphicsFromHDC, hdc, uintptr(unsafe.Pointer(&g))) != 0 {
		return
	}
	defer call(procGdipDeleteGraphics, g)
	call(procGdipSetSmoothingMode, g, 4)
	path := roundPath(r, radius)
	if path == 0 {
		return
	}
	defer call(procGdipDeletePath, path)
	if stroke {
		var p uintptr
		if call(procGdipCreatePen1, argb(clr, alpha), uintptr(0x3f800000), 2, uintptr(unsafe.Pointer(&p))) == 0 {
			call(procGdipDrawPath, g, p, path)
			call(procGdipDeletePen, p)
		}
	} else {
		var b uintptr
		if call(procGdipCreateSolidFill, argb(clr, alpha), uintptr(unsafe.Pointer(&b))) == 0 {
			call(procGdipFillPath, g, b, path)
			call(procGdipDeleteBrush, b)
		}
	}
}
func fillRound(hdc uintptr, r rect, clr uint32, rounding int32) {
	paintRounded(hdc, r, clr, rounding/2, 255, false)
}
func translucentRound(hdc uintptr, r rect, clr uint32, radius int32, alpha uint8) {
	paintRounded(hdc, r, clr, radius, alpha, false)
}
func outlineRound(hdc uintptr, r rect, clr uint32, radius int32, alpha uint8) {
	paintRounded(hdc, r, clr, radius, alpha, true)
}
func softCircle(hdc uintptr, x, y, diameter int32, clr uint32, alpha uint8) {
	if a == nil || a.gdiToken == 0 {
		return
	}
	var g, b uintptr
	if call(procGdipCreateGraphicsFromHDC, hdc, uintptr(unsafe.Pointer(&g))) != 0 {
		return
	}
	defer call(procGdipDeleteGraphics, g)
	call(procGdipSetSmoothingMode, g, 4)
	if call(procGdipCreateSolidFill, argb(clr, alpha), uintptr(unsafe.Pointer(&b))) == 0 {
		call(procGdipFillEllipseI, g, b, uintptr(x), uintptr(y), uintptr(diameter), uintptr(diameter))
		call(procGdipDeleteBrush, b)
	}
}

func createControl(class, text string, style uint32, x, y, w, h int32, id int) uintptr {
	return call(procCreateWindow, 0, uintptr(unsafe.Pointer(ptr(class))), uintptr(unsafe.Pointer(ptr(text))), uintptr(style|wsChild|wsVisible), uintptr(a.px(x)), uintptr(a.px(y)), uintptr(a.px(x+w)-a.px(x)), uintptr(a.px(y+h)-a.px(y)), a.hwnd, uintptr(id), 0, 0)
}
func (ap *app) button(id int, title string, x, y, w, h int32, clr, fg uint32) {
	if ap.controlRects == nil {
		ap.controlRects = make(map[int]rect)
	}
	ap.controlRects[id] = rect{0, 0, w, h}
	handle := createControl("BUTTON", title, bsOwnerDraw|wsTabstop, x, y, w, h, id)
	ap.buttons[id] = handle
	ap.styles[uint32(id)] = style{title, clr, fg}
	ap.enabled[id] = true
}
func (ap *app) enqueue(e event) {
	select {
	case <-ap.done:
		return
	default:
	}
	ap.mu.Lock()
	notify := len(ap.queue) == 0
	ap.queue = append(ap.queue, e)
	ap.mu.Unlock()
	if notify {
		call(procPostMessage, ap.hwnd, wmAppEvent, 0, 0)
	}
}

// Drain a burst once. A traffic event cannot invalidate the hero or any button.
func (ap *app) update() {
	ap.mu.Lock()
	arr := ap.queue
	ap.queue = nil
	ap.mu.Unlock()
	if len(arr) == 0 {
		return
	}
	change := ap.uiState.applyEvents(arr)
	for _, e := range arr {
		if e.line != "" {
			ap.logLine(e.line)
		}
		if e.setPath != "" {
			ap.path = e.setPath
		}
	}
	if change.saveConfig {
		ap.saveConfig()
	}
	switch change.redraw {
	case redrawContent:
		ap.updateButtons()
		ap.invalidate(nil)
	case redrawTraffic:
		ap.invalidate(&networkRect)
	}
}

func (ap *app) invalidate(r *rect) {
	if ap.hwnd == 0 || call(procIsWindowVisible, ap.hwnd) == 0 {
		return
	}
	var rp uintptr
	var physical rect
	if r != nil {
		physical = ap.scaledRect(*r)
		rp = uintptr(unsafe.Pointer(&physical))
	}
	call(procInvalidateRect, ap.hwnd, rp, 0)
}

var networkRect = rect{240, 564, 1054, 686}

func boolean(v bool) uintptr {
	if v {
		return 1
	}
	return 0
}
func (ap *app) logLine(s string) {
	date := time.Now().Format("15:04:05")
	line := "[" + date + "] " + s + "\r\n"
	f, err := os.OpenFile(filepath.Join(ap.dataDir, "session.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err == nil {
		_, _ = io.WriteString(f, line)
		_ = f.Close()
	}
	call(procSendMessage, ap.logEdit, emSetSel, uintptr(^uint32(0)), uintptr(^uint32(0)))
	call(procSendMessage, ap.logEdit, emReplaceSel, 0, uintptr(unsafe.Pointer(ptr(line))))
}
func (ap *app) work(message string, fn func() error) {
	if ap.busy {
		return
	}
	ap.busy = true
	ap.attemptingProfile = ""
	ap.updateButtons()
	ap.invalidate(nil)
	ap.logLine(message)
	go func() {
		err := fn()
		if err != nil {
			ap.enqueue(event{line: "HATA: " + err.Error(), state: "İşlem başarısız"})
		}
		f := false
		ap.enqueue(event{busy: &f})
	}()
}
func (ap *app) updateButtons() {
	for id, h := range ap.buttons {
		enabled := !ap.busy || id == idOpenDiscord || id == idOpenLog
		if ap.enabled[id] != enabled {
			ap.enabled[id] = enabled
			call(procEnableWindow, h, boolean(enabled))
		}
	}
	ap.refreshStartButton()
}
func (ap *app) refreshStartButton() {
	title := ap.L("Bağlantıyı Başlat", "Start connection")
	clr := primary
	if ap.busy {
		title = ap.L("İşlem sürüyor...", "Working...")
		clr = primarySoft
	} else if ap.connected || ap.running {
		title = ap.L("Bağlantıyı Durdur", "Stop connection")
		clr = red
	}
	if old, ok := ap.styles[idStart]; ok && old != (style{title, clr, white}) {
		ap.styles[idStart] = style{title, clr, white}
		call(procSetWindowText, ap.buttons[idStart], uintptr(unsafe.Pointer(ptr(title))))
		if h := ap.buttons[idStart]; h != 0 {
			call(procInvalidateRect, h, 0, 0)
		}
	}
}
func (ap *app) selectProfile(id string) {
	if ap.busy || (id == "auto" && ap.mode == "auto") || (ap.mode == "manual" && ap.selectedProfile == id) {
		return
	}
	if id == "auto" {
		ap.mode = "auto"
	} else {
		ap.mode = "manual"
		ap.selectedProfile = id
	}
	ap.saveConfig()
	ap.logLine(ap.L("Profil seçimi kaydedildi: ", "Profile selection saved: ") + ap.profileName(id))
	ap.invalidate(nil)
	for _, n := range profileButtonIDs {
		if h := ap.buttons[n]; h != 0 {
			call(procInvalidateRect, h, 0, 0)
		}
	}
}

func (ap *app) confirm(title, body string) bool {
	return call(procMessageBox, ap.hwnd, uintptr(unsafe.Pointer(ptr(body))), uintptr(unsafe.Pointer(ptr(title))), mbYesNo|mbIconQuestion) == idYes
}
func (ap *app) alert(title, body string) {
	call(procMessageBox, ap.hwnd, uintptr(unsafe.Pointer(ptr(body))), uintptr(unsafe.Pointer(ptr(title))), mbIconWarning)
}

func (ap *app) browse() {
	file := make([]uint16, 2048)
	if ap.path != "" {
		copy(file, syscall.StringToUTF16(ap.path))
	}
	filter := make([]uint16, 0, 90)
	for _, part := range []string{"GoodbyeDPI (goodbyedpi.exe)", "goodbyedpi.exe", "EXE dosyaları", "*.exe", ""} {
		filter = append(filter, utf16.Encode([]rune(part))...)
		filter = append(filter, 0)
	}
	ofn := openFileName{hwnd: ap.hwnd, filter: &filter[0], file: &file[0], maxFile: uint32(len(file)), title: ptr("GoodbyeDPI uygulamasını seç"), flags: ofnFileMustExist | ofnPathMustExist | ofnExplorer}
	ofn.size = uint32(unsafe.Sizeof(ofn))
	if call(procGetOpenFileName, uintptr(unsafe.Pointer(&ofn))) != 0 {
		p := syscall.UTF16ToString(file)
		if !strings.EqualFold(filepath.Base(p), "goodbyedpi.exe") {
			ap.alert("Yanlış dosya", "Lütfen goodbyedpi.exe dosyasını seç.")
			return
		}
		ap.path = p
		ap.saveConfig()
		call(procSetWindowText, ap.pathEdit, uintptr(unsafe.Pointer(ptr(p))))
		ap.logLine("GoodbyeDPI seçildi: " + p)
	}
}

// Closing the main window hides it in the Windows notification area. We do not
// kill the external DPI process: it remains connected just like when visible.
func (ap *app) notifyData() notifyIconData {
	n := notifyIconData{Hwnd: ap.hwnd, ID: trayIconID, Flags: nifMessage | nifIcon | nifTip, CallbackMessage: wmTray, Icon: ap.trayIcon}
	n.Size = uint32(unsafe.Sizeof(n))
	copy(n.Tip[:], syscall.StringToUTF16("Discord Bridge - bağlantı arka planda çalışıyor"))
	return n
}
func (ap *app) addTray() bool {
	if ap.trayAdded {
		return true
	}
	n := ap.notifyData()
	ap.trayAdded = call(procShellNotifyIcon, nimAdd, uintptr(unsafe.Pointer(&n))) != 0
	return ap.trayAdded
}
func (ap *app) removeTray() {
	if !ap.trayAdded {
		return
	}
	n := ap.notifyData()
	call(procShellNotifyIcon, nimDelete, uintptr(unsafe.Pointer(&n)))
	ap.trayAdded = false
}
func (ap *app) hideToTray() {
	if !ap.addTray() {
		// A failed shell call must not strand the user with an invisible window.
		ap.alert("Tepsi simgesi oluşturulamadı", "Windows bildirim alanına simge eklenemedi. Pencere açık kalacak.")
		return
	}
	call(procShowWindow, ap.hwnd, swHide)
	if !ap.trayHintShown {
		ap.logLine("Pencere sistem tepsisine gizlendi. Simgesine çift tıklayarak geri açabilirsiniz.")
		ap.trayHintShown = true
	}
}
func (ap *app) restoreFromTray() {
	call(procShowWindow, ap.hwnd, swRestore)
	call(procSetForeground, ap.hwnd)
}
func (ap *app) trayMenu() {
	menu := call(procCreatePopupMenu)
	if menu == 0 {
		return
	}
	defer call(procDestroyMenu, menu)
	call(procAppendMenu, menu, mfString, trayMenuShow, uintptr(unsafe.Pointer(ptr(ap.L("Pencereyi Göster", "Show Window")))))
	call(procAppendMenu, menu, mfString, trayMenuDiscord, uintptr(unsafe.Pointer(ptr(ap.L("Discord'u Aç", "Open Discord")))))
	call(procAppendMenu, menu, mfSeparator, 0, 0)
	call(procAppendMenu, menu, mfString, trayMenuExit, uintptr(unsafe.Pointer(ptr(ap.L("Programdan Çık (DPI Açık Kalır)", "Exit (Keep DPI Running)")))))
	call(procAppendMenu, menu, mfString, trayMenuStopExit, uintptr(unsafe.Pointer(ptr(ap.L("DPI'ı Durdur ve Çık", "Stop DPI and Exit")))))
	p := point{}
	if call(procGetCursorPos, uintptr(unsafe.Pointer(&p))) == 0 {
		return
	}
	// Windows requires foreground activation to dismiss a popup on an outside click.
	call(procSetForeground, ap.hwnd)
	selected := call(procTrackPopupMenu, menu, tpmRightButton|tpmReturnCmd, uintptr(p.x), uintptr(p.y), 0, ap.hwnd, 0)
	switch selected {
	case trayMenuShow:
		ap.restoreFromTray()
	case trayMenuDiscord:
		ap.openDiscord()
	case trayMenuExit:
		// Close only the UI; external GoodbyeDPI continues if it is active.
		call(procDestroyWindow, ap.hwnd)
	case trayMenuStopExit:
		if ap.busy {
			ap.restoreFromTray()
			ap.alert("İşlem devam ediyor", "Devam eden bağlantı işlemi bitince yeniden deneyin.")
			return
		}
		if err := ap.stopOwned(); err != nil {
			ap.restoreFromTray()
			ap.alert("DPI durdurulamadı", err.Error())
			return
		}
		call(procDestroyWindow, ap.hwnd)
	}
}

func panelLabel(hdc uintptr, text string, x, y, w int32, colorValue uint32, font uintptr) {
	drawText(hdc, text, x, y, w, 21, colorValue, font, dtSingleLine|dtVCenter)
}
func lightDot(hdc uintptr, x, y int32, clr uint32) {
	brush := call(procCreateSolidBrush, uintptr(clr))
	old := call(procSelectObject, hdc, brush)
	pen := call(procCreatePen, 0, 1, uintptr(clr))
	oldp := call(procSelectObject, hdc, pen)
	call(procEllipse, hdc, uintptr(x), uintptr(y), uintptr(x+10), uintptr(y+10))
	call(procSelectObject, hdc, oldp)
	call(procSelectObject, hdc, old)
	call(procDeleteObject, pen)
	call(procDeleteObject, brush)
}

func drawNetworkChart(hdc uintptr, rx, tx []float64, x, y, w, h int32) {
	grid := call(procCreatePen, 0, 1, uintptr(edge))
	old := call(procSelectObject, hdc, grid)
	for n := int32(0); n < 4; n++ {
		yy := y + 8 + n*24
		call(procMoveToEx, hdc, uintptr(x), uintptr(yy), 0)
		call(procLineTo, hdc, uintptr(x+w), uintptr(yy))
	}
	call(procSelectObject, hdc, old)
	call(procDeleteObject, grid)
	both := [][]float64{rx, tx}
	colors := []uint32{teal, color(158, 168, 255)}
	maxv := 10.0
	for _, series := range both {
		for _, v := range series {
			if v > maxv {
				maxv = v
			}
		}
	}
	for j, series := range both {
		if len(series) < 2 {
			continue
		}
		pen := call(procCreatePen, 0, 2, uintptr(colors[j]))
		old = call(procSelectObject, hdc, pen)
		denom := len(series) - 1
		for i, v := range series {
			xx := x + int32(i)*w/int32(denom)
			yy := y + h - 6 - int32((v/maxv)*float64(h-14))
			if i == 0 {
				call(procMoveToEx, hdc, uintptr(xx), uintptr(yy), 0)
			} else {
				call(procLineTo, hdc, uintptr(xx), uintptr(yy))
			}
		}
		call(procSelectObject, hdc, old)
		call(procDeleteObject, pen)
	}
}
func brighten(c uint32, adjust int) uint32 {
	f := func(v uint32) uint32 {
		n := int(v) + adjust
		if n < 0 {
			n = 0
		}
		if n > 255 {
			n = 255
		}
		return uint32(n)
	}
	return color(f(c&255), f((c>>8)&255), f((c>>16)&255))
}

func (ap *app) refreshLabels() {
	labels := map[int]string{
		idOpenDiscord: ap.L("Discord’u Aç", "Open Discord"),
		idDNS:         ap.L("DNS ayarları", "DNS settings"),
		idRestore:     ap.L("DNS geri al", "Restore DNS"),
		idTest:        ap.L("Bağlantı testi", "Connection test"),
		idOpenLog:     ap.L("İşlem günlüğü", "Activity log"),
		idPack:        ap.L("Motoru doğrula", "Verify engine"),
		idLanguage:    ap.L("Türkçe  /  English", "English  /  Türkçe"),
	}
	for i, id := range profileButtonIDs {
		labels[id] = ap.profileName(profileIDs[i])
	}
	for id, label := range labels {
		if v, ok := ap.styles[uint32(id)]; ok {
			v.caption = label
			ap.styles[uint32(id)] = v
		}
		if h := ap.buttons[id]; h != 0 {
			call(procSetWindowText, h, uintptr(unsafe.Pointer(ptr(label))))
		}
		if h := ap.buttons[id]; h != 0 {
			call(procInvalidateRect, h, 0, 0)
		}
	}
	ap.refreshStartButton()
}

func wndproc(hwnd uintptr, msg uint32, wparam, lparam uintptr) uintptr {
	switch msg {
	case wmPaint:
		ps := paintStruct{}
		h := call(procBeginPaint, hwnd, uintptr(unsafe.Pointer(&ps)))
		a.paintBuffered(h, ps.paint)
		call(procEndPaint, hwnd, uintptr(unsafe.Pointer(&ps)))
		return 0
	case wmEraseBkgnd:
		return 1
	case wmSetCursor:
		newHover := 0
		for id, btn := range a.buttons {
			if btn == wparam {
				newHover = id
				break
			}
		}
		if newHover != a.hoverID {
			prev := a.hoverID
			a.hoverID = newHover
			if v := a.buttons[prev]; v != 0 {
				call(procInvalidateRect, v, 0, 0)
			}
			if v := a.buttons[newHover]; v != 0 {
				call(procInvalidateRect, v, 0, 0)
			}
		}
		if newHover != 0 && a.enabled[newHover] {
			call(procSetCursor, call(procLoadCursor, 0, 32649))
			return 1
		}
	case wmDrawItem:
		if lparam != 0 {
			a.paintButton((*drawItem)(unsafe.Pointer(lparam)))
		}
		return 1
	case wmCtlColorEdit, wmCtlColorStatic:
		call(procSetTextColor, wparam, uintptr(white))
		call(procSetBkColor, wparam, uintptr(inputBG))
		return a.brushEdit
	case wmCommand:
		switch int(wparam & 0xffff) {
		case idBrowse:
			a.browse()
		case idStart:
			if a.busy {
				break
			}
			if a.connected || a.running {
				a.handleStop()
			} else {
				a.handleStart()
			}
		case idStop:
			a.handleStop()
		case idDNS:
			a.handleDNS()
		case idRestore:
			a.handleRestore()
		case idTest:
			a.handleTest()
		case idLanguage:
			if a.language == "tr" {
				a.language = "en"
			} else {
				a.language = "tr"
			}
			a.saveConfig()
			a.refreshLabels()
			a.invalidate(nil)
		case idProfileAuto, idProfileSpecial, idProfile5, idProfile6, idProfile9, idProfile2:
			for j, n := range profileButtonIDs {
				if n == int(wparam&0xffff) {
					a.selectProfile(profileIDs[j])
					break
				}
			}
		case idPack:
			a.handleVerifyMotor()
		case idOpenDiscord:
			a.logLine("Kullanıcı Discord'u Aç düğmesine bastı.")
			a.openDiscord()
		case idOpenLog:
			logPath := filepath.Join(a.dataDir, "session.log")
			call(procShellExecute, hwnd, uintptr(unsafe.Pointer(ptr("open"))), uintptr(unsafe.Pointer(ptr(logPath))), 0, 0, swNormal)
		}
		return 0
	case wmAppEvent:
		a.update()
		return 0
	case wmClose:
		a.hideToTray()
		return 0
	case wmTray:
		switch uint32(lparam) {
		case wmLButtonDblClk:
			a.restoreFromTray()
		case wmRButtonUp, wmContextMenu:
			a.trayMenu()
		}
		return 0
	case wmTrayRestore:
		a.restoreFromTray()
		return 0
	case wmDestroy:
		close(a.done)
		a.releaseBuffer()
		a.removeTray()
		for _, h := range []uintptr{a.fTitle, a.fHeading, a.fText, a.fSmall, a.fBtn, a.fHero, a.fTiny, a.brushBG, a.brushCard, a.brushEdit, a.brushSidebar} {
			if h != 0 {
				call(procDeleteObject, h)
			}
		}
		if a.gdiToken != 0 {
			call(procGdiplusShutdown, a.gdiToken)
		}
		call(procPostQuitMessage, 0)
		return 0
	}
	return call(procDefWindowProc, hwnd, uintptr(msg), wparam, lparam)
}

func stateDir() string {
	home := os.Getenv("LOCALAPPDATA")
	if home == "" {
		home, _ = os.UserConfigDir()
	}
	p := filepath.Join(home, "DiscordBridge")
	os.MkdirAll(p, 0700)
	return p
}
func findGoodbyeDPI(root string) string {
	candidates := []string{
		filepath.Join(root, "x86_64", "goodbyedpi.exe"),
		filepath.Join(stateDir(), "engine", "goodbyedpi.exe"),
		filepath.Join(stateDir(), "engine-offline", "goodbyedpi.exe"),
		filepath.Join(root, "GoodbyeDPI", "x86_64", "goodbyedpi.exe"),
		filepath.Join(root, "goodbyedpi.exe"),
		filepath.Join(filepath.Dir(root), "x86_64", "goodbyedpi.exe"),
		`C:\GoodbyeDPI\x86_64\goodbyedpi.exe`,
		`C:\GoodbyeDPI\goodbyedpi.exe`,
	}
	for _, p := range candidates {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
	}
	// Common extracted GoodbyeDPI folders; never recurse through the full disk.
	home, _ := os.UserHomeDir()
	for _, dir := range []string{filepath.Join(home, "Downloads"), filepath.Join(home, "Desktop"), filepath.Join(home, "Documents")} {
		for _, pattern := range []string{"*goodbyedpi*\\x86_64\\goodbyedpi.exe", "*GoodbyeDPI*\\x86_64\\goodbyedpi.exe", "*GoodbyeDPI*\\goodbyedpi.exe"} {
			found, _ := filepath.Glob(filepath.Join(dir, pattern))
			for _, f := range found {
				if checkEngineFiles(f) == nil {
					return f
				}
			}
		}
	}
	return ""
}
func (ap *app) saveConfig() {
	data, _ := json.MarshalIndent(config{ExePath: ap.path, Mode: ap.mode, SelectedProfile: ap.selectedProfile, LastWorkingProfile: ap.lastWorkingProfile, Language: ap.language}, "", "  ")
	_ = os.WriteFile(filepath.Join(ap.dataDir, "settings.json"), data, 0600)
}
func (ap *app) loadConfig() {
	data, err := os.ReadFile(filepath.Join(ap.dataDir, "settings.json"))
	if err == nil {
		var c config
		if json.Unmarshal(data, &c) == nil {
			if c.Mode == "manual" || c.Mode == "auto" {
				ap.mode = c.Mode
			}
			if profileValid(c.SelectedProfile) {
				ap.selectedProfile = c.SelectedProfile
			}
			if profileValid(c.LastWorkingProfile) {
				ap.lastWorkingProfile = c.LastWorkingProfile
			}
			if c.Language == "en" || c.Language == "tr" {
				ap.language = c.Language
			}
			if _, err := os.Stat(c.ExePath); err == nil {
				ap.path = c.ExePath
			}
		}
	}
}
func powershell(script string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.CombinedOutput()
	value := strings.TrimSpace(string(bytes.TrimPrefix(out, []byte{0xEF, 0xBB, 0xBF})))
	if err != nil {
		if len(value) > 350 {
			value = value[len(value)-350:]
		}
		return value, fmt.Errorf("PowerShell: %s", value)
	}
	return value, nil
}
func escapePS(s string) string      { return strings.ReplaceAll(s, "'", "''") }
func (ap *app) backupPath() string  { return filepath.Join(ap.dataDir, "dns-backup.json") }
func (ap *app) processPath() string { return filepath.Join(ap.dataDir, "process.json") }
func (ap *app) domainsPath() string { return filepath.Join(ap.dataDir, "discord-domains.txt") }
func (ap *app) createDomains() error {
	content := "discord.com\r\ndiscord.gg\r\ndiscordapp.com\r\ndiscordapp.net\r\ndiscord.media\r\n"
	return os.WriteFile(ap.domainsPath(), []byte(content), 0600)
}
func processRunning(pid int, expected string) bool {
	if pid < 1 {
		return false
	}
	script := fmt.Sprintf("$p=Get-CimInstance Win32_Process -Filter 'ProcessId = %d' -ErrorAction SilentlyContinue; if($p -and $p.ExecutablePath -and ($p.ExecutablePath -ieq '%s')) { 'YES' } else { 'NO' }", pid, escapePS(expected))
	out, e := powershell(script)
	return e == nil && strings.TrimSpace(out) == "YES"
}
func (ap *app) readProcess() (processInfo, error) {
	var data processInfo
	b, e := os.ReadFile(ap.processPath())
	if e != nil {
		return data, e
	}
	e = json.Unmarshal(b, &data)
	return data, e
}
func (ap *app) isRunning() bool {
	p, e := ap.readProcess()
	return e == nil && processRunning(p.PID, p.Path)
}

// Explicit Start: check DNS and HTTPS, then try saved success or the special profile
// first; fall back deterministically only in auto mode. Never auto-open Discord.
func (ap *app) handleStart() {
	plan := ap.profilePlan() // capture on UI thread
	manual := ap.mode == "manual"
	ap.work(ap.L("Bağlantı işlemi başladı. ExitLag ayarları değiştirilmeyecek.", "Starting connection. ExitLag settings remain unchanged."), func() error {
		if err := ap.createDomains(); err != nil {
			return err
		}
		if ap.isRunning() {
			p, _ := ap.readProcess()
			if err := checkHTTPS(); err == nil {
				t := true
				id := profileIDFromName(p.Profile)
				if id == "" {
					id = "special"
				}
				ap.enqueue(event{running: &t, connected: &t, activeProfile: id, state: "Bağlantı hazır", line: ap.L("Önceki bağlantı hâlâ çalışıyor.", "Previous connection is still running.")})
				return nil
			}
			if err := ap.stopOwned(); err != nil {
				return err
			}
		}
		ap.enqueue(event{state: "Ağ kontrol ediliyor", line: ap.L("DNS kontrol ediliyor...", "Checking DNS...")})
		ips, err := net.LookupIP("updates.discord.com")
		poisoned := err != nil || len(ips) == 0
		for _, ip := range ips {
			if ip.String() == "195.175.254.2" || strings.EqualFold(ip.String(), "2a01:358:4014:a00::3") {
				poisoned = true
			}
		}
		if poisoned {
			ap.enqueue(event{state: "DNS düzeltiliyor", line: ap.L("DNS yanıtı hatalı. Yedek alınıp Cloudflare deneniyor.", "DNS response blocked. Backing up and applying Cloudflare.")})
			if err := ap.ensureDNS(); err != nil {
				return fmt.Errorf("DNS: %w", err)
			}
		} else {
			ap.enqueue(event{dns: "DNS doğru", line: ap.L("DNS yanıtı normal.", "DNS resolution looks normal.")})
		}
		if err := checkHTTPS(); err == nil {
			t := true
			ap.enqueue(event{connected: &t, activeProfile: "direct", state: "Bağlantı hazır", line: ap.L("Doğrudan erişim başarılı; DPI motoru gerekmedi.", "Direct access works; DPI engine was not required.")})
			return nil
		}
		if err := ap.ensureDependency(); err != nil {
			return err
		}
		failed := make([]string, 0, len(plan))
		for _, pr := range plan {
			ap.enqueue(event{attempt: pr.id, state: "Profiller deneniyor", line: ap.L("Denenen profil: ", "Testing profile: ") + ap.profileName(pr.id)})
			if err := ap.launchDPI(pr.id, pr.args); err != nil {
				failed = append(failed, pr.id+": "+err.Error())
				continue
			}
			time.Sleep(1200 * time.Millisecond)
			if err := checkHTTPS(); err == nil {
				t := true
				ap.enqueue(event{running: &t, connected: &t, activeProfile: pr.id, lastWorking: pr.id, state: "Bağlantı hazır", line: ap.L("Çalışan profil kaydedildi: ", "Working profile saved: ") + ap.profileName(pr.id)})
				return nil
			} else {
				failed = append(failed, pr.id+": "+err.Error())
			}
			_ = ap.stopOwned()
		}
		f := false
		ap.enqueue(event{connected: &f, activeProfile: "none", state: "İşlem başarısız"})
		if manual {
			return fmt.Errorf("selected profile did not pass HTTPS test: %s", strings.Join(failed, " | "))
		}
		return fmt.Errorf("all profiles failed HTTPS test: %s", strings.Join(failed, " | "))
	})
}
func profileIDFromName(s string) string {
	if profileValid(s) {
		return s
	}
	if strings.Contains(strings.ToLower(s), "exitlag") {
		return "special"
	}
	if strings.Contains(s, "(-6)") {
		return "alt6"
	}
	if strings.Contains(s, "(-9)") {
		return "alt9"
	}
	if strings.Contains(s, "(-2)") {
		return "alt2"
	}
	if strings.Contains(s, "(-5)") {
		return "standard5"
	}
	return ""
}

func (ap *app) openDiscord() {
	// Prefer the installed desktop application; browser remains a fallback.
	r := call(procShellExecute, ap.hwnd, uintptr(unsafe.Pointer(ptr("open"))), uintptr(unsafe.Pointer(ptr("discord://"))), 0, 0, swNormal)
	if r <= 32 {
		call(procShellExecute, ap.hwnd, uintptr(unsafe.Pointer(ptr("open"))), uintptr(unsafe.Pointer(ptr("https://discord.com/app"))), 0, 0, swNormal)
	}
}

func (ap *app) ensureDNS() error {
	d, e := readCurrentDNS()
	if e != nil {
		return e
	}
	backup := ap.backupPath()
	if original, e := os.ReadFile(backup); e == nil {
		var b dnsBackup
		if json.Unmarshal(original, &b) != nil || !strings.EqualFold(b.GUID, d.GUID) {
			return errors.New("Önceki DNS yedeği başka bağdaştırıcıya ait; mevcut ayarlar değiştirilmedi")
		}
	} else {
		b, _ := json.MarshalIndent(d, "", "  ")
		if e := os.WriteFile(backup, b, 0600); e != nil {
			return e
		}
	}
	v6 := ""
	if d.IPv6Enabled {
		v6 = ",'2606:4700:4700::1111','2606:4700:4700::1001'"
	}
	script := fmt.Sprintf("$ErrorActionPreference='Stop'; Set-DnsClientServerAddress -InterfaceIndex %d -ServerAddresses @('1.1.1.1','1.0.0.1'%s);Clear-DnsClientCache; 'DNS_OK'", d.Index, v6)
	if _, e := powershell(script); e != nil {
		return e
	}
	ap.enqueue(event{line: "Cloudflare DNS uygulandı, eski DNS ayarları yedeklendi: " + d.Alias, dns: "Cloudflare"})
	return nil
}

func (ap *app) dependencyRoot() string { return filepath.Join(ap.dataDir, "engine-offline") }
func (ap *app) ensureDependency() error {
	// This edition is fully self-contained: never download or use a different local motor.
	zipData, e := readOwnPayload()
	if e != nil {
		return fmt.Errorf("Gömülü motor bozuk veya eksik: %w", e)
	}
	ap.enqueue(event{line: "Dahili GoodbyeDPI / WinDivert motoru SHA256 ile doğrulandı."})
	if e = extractPayload(zipData, ap.dependencyRoot()); e != nil {
		return e
	}
	exe := filepath.Join(ap.dependencyRoot(), "goodbyedpi.exe")
	ap.enqueue(event{setPath: exe, line: "Motor uygulamaya gömülü. Harici indirme yapılmadı."})
	return nil
}

func checkEngineFiles(exe string) error {
	if !strings.EqualFold(filepath.Base(exe), "goodbyedpi.exe") {
		return errors.New("program dosyasının adı beklenenden farklı")
	}
	for _, name := range []string{"goodbyedpi.exe", "WinDivert.dll", "WinDivert64.sys"} {
		path := filepath.Join(filepath.Dir(exe), name)
		f, e := os.Open(path)
		if e != nil {
			return fmt.Errorf("%s bulunamadı", name)
		}
		b := make([]byte, 2)
		_, e = io.ReadFull(f, b)
		f.Close()
		if e != nil || string(b) != "MZ" {
			return fmt.Errorf("%s geçersiz Windows dosyası", name)
		}
	}
	return nil
}

//go:embed GOODBYEDPI_APACHE_2_LICENSE.txt WINDIVERT_LGPL_3_LICENSE.txt
var licenseAssets embed.FS

// Payload format: [ZIP][MAGIC 8 bytes][ZIP length little-endian uint64][ZIP SHA256 32 bytes].
// The last 48 bytes permit deterministic verification before extraction.
func readOwnPayload() ([]byte, error) {
	own, e := os.Executable()
	if e != nil {
		return nil, e
	}
	f, e := os.Open(own)
	if e != nil {
		return nil, e
	}
	defer f.Close()
	size, e := f.Seek(0, io.SeekEnd)
	if e != nil {
		return nil, e
	}
	if size < 48 {
		return nil, errors.New("Gömülü motor yok")
	}

	// Unsigned releases carry the trailer at EOF. Authenticode-signed releases
	// add WIN_CERTIFICATE after it; its start is the PE security directory's
	// FILE OFFSET (not an RVA). Locate the trailer immediately before that.
	offsets := []int64{size - 48}
	header := make([]byte, 4096)
	n, _ := f.ReadAt(header, 0)
	header = header[:n]
	if n >= 0x40 && string(header[:2]) == "MZ" {
		peOffset := int(binary.LittleEndian.Uint32(header[0x3c:0x40]))
		optOffset := peOffset + 24
		// This app is compiled for 64-bit Windows (PE32+).
		if optOffset+152 <= n && string(header[peOffset:peOffset+4]) == "PE\x00\x00" &&
			binary.LittleEndian.Uint16(header[optOffset:optOffset+2]) == 0x20b {
			// IMAGE_DIRECTORY_ENTRY_SECURITY = 4, offset +112 for PE32+.
			securityEntry := optOffset + 112 + 4*8
			certificateOffset := int64(binary.LittleEndian.Uint32(header[securityEntry : securityEntry+4]))
			certificateSize := int64(binary.LittleEndian.Uint32(header[securityEntry+4 : securityEntry+8]))
			if certificateSize > 0 && certificateOffset > 48 && certificateOffset <= size {
				for padding := int64(0); padding <= 16; padding++ {
					offsets = append(offsets, certificateOffset-48-padding)
				}
			}
		}
	}
	for _, trailerOffset := range offsets {
		if trailerOffset < 0 || trailerOffset+48 > size {
			continue
		}
		trailer := make([]byte, 48)
		if _, e = f.ReadAt(trailer, trailerOffset); e != nil {
			continue
		}
		if string(trailer[:8]) != payloadMagic {
			continue
		}
		length := binary.LittleEndian.Uint64(trailer[8:16])
		if length == 0 || length > 40<<20 || int64(length) > trailerOffset {
			continue
		}
		zipData := make([]byte, int(length))
		if _, e = f.ReadAt(zipData, trailerOffset-int64(length)); e != nil {
			continue
		}
		sum := sha256.Sum256(zipData)
		if bytes.Equal(sum[:], trailer[16:48]) {
			return zipData, nil
		}
	}
	return nil, errors.New("Gömülü motor yok veya veri bütünlüğü bozulmuş")
}

func readEngineZIP(b []byte) (map[string][]byte, error) {
	z, e := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if e != nil {
		return nil, e
	}
	names := map[string]bool{"goodbyedpi.exe": true, "WinDivert.dll": true, "WinDivert64.sys": true}
	content := map[string][]byte{}
	for _, f := range z.File {
		if !names[f.Name] {
			continue
		}
		if f.UncompressedSize64 > 20<<20 {
			return nil, errors.New("Beklenmeyen büyüklükte motor dosyası")
		}
		r, e := f.Open()
		if e != nil {
			return nil, e
		}
		data, e := io.ReadAll(io.LimitReader(r, 20<<20))
		_ = r.Close()
		if e != nil {
			return nil, e
		}
		if len(data) < 2 || string(data[:2]) != "MZ" {
			return nil, fmt.Errorf("%s PE dosyası değil", f.Name)
		}
		content[f.Name] = data
	}
	if len(content) != 3 {
		return nil, errors.New("Gömülü motor üç gerekli Windows dosyasını içermiyor")
	}
	return content, nil
}
func extractPayload(b []byte, root string) error {
	content, e := readEngineZIP(b)
	if e != nil {
		return e
	}
	if e = os.MkdirAll(root, 0700); e != nil {
		return e
	}
	for name, data := range content {
		path := filepath.Join(root, name)
		if old, e := os.ReadFile(path); e == nil && bytes.Equal(old, data) {
			continue
		}
		tmp := path + ".new"
		if e = os.WriteFile(tmp, data, 0600); e != nil {
			return e
		}
		// Windows does not always permit replacing an existing destination with Rename.
		if _, statErr := os.Stat(path); statErr == nil {
			if err := os.Remove(path); err != nil {
				_ = os.Remove(tmp)
				return fmt.Errorf("%s eski sürümü kaldırılamadı: %w", name, err)
			}
		}
		if e = os.Rename(tmp, path); e != nil {
			_ = os.Remove(tmp)
			return fmt.Errorf("%s çıkarılamadı: %w", name, e)
		}
	}
	// Expose open-source license texts next to the extracted components.
	for _, entry := range []struct{ dst, src string }{
		{"LICENSE-GoodbyeDPI-Apache-2.0.txt", "GOODBYEDPI_APACHE_2_LICENSE.txt"},
		{"LICENSE-WinDivert-LGPL-3.txt", "WINDIVERT_LGPL_3_LICENSE.txt"},
	} {
		data, e := licenseAssets.ReadFile(entry.src)
		if e == nil {
			_ = os.WriteFile(filepath.Join(root, entry.dst), data, 0600)
		}
	}
	return checkEngineFiles(filepath.Join(root, "goodbyedpi.exe"))
}
func makePayload(exe string) ([]byte, error) {
	if e := checkEngineFiles(exe); e != nil {
		return nil, e
	}
	var buf bytes.Buffer
	z := zip.NewWriter(&buf)
	for _, name := range []string{"goodbyedpi.exe", "WinDivert.dll", "WinDivert64.sys"} {
		raw, e := os.ReadFile(filepath.Join(filepath.Dir(exe), name))
		if e != nil {
			return nil, e
		}
		w, e := z.Create(name)
		if e != nil {
			return nil, e
		}
		if _, e = w.Write(raw); e != nil {
			return nil, e
		}
	}
	for _, entry := range []struct{ dst, src string }{
		{"LICENSE-GoodbyeDPI-Apache-2.0.txt", "GOODBYEDPI_APACHE_2_LICENSE.txt"},
		{"LICENSE-WinDivert-LGPL-3.txt", "WINDIVERT_LGPL_3_LICENSE.txt"},
	} {
		data, e := licenseAssets.ReadFile(entry.src)
		if e != nil {
			return nil, e
		}
		w, e := z.Create(entry.dst)
		if e != nil {
			return nil, e
		}
		if _, e = w.Write(data); e != nil {
			return nil, e
		}
	}
	// Unmodified dependencies. The corresponding original-source license obligations remain with distributors.
	w, e := z.Create("ATTRIBUTION.txt")
	if e != nil {
		return nil, e
	}
	_, e = w.Write([]byte("GoodbyeDPI (Apache-2.0): https://github.com/ValdikSS/GoodbyeDPI\r\nWinDivert (LGPL-3): https://github.com/basil00/WinDivert\r\nThe original WinDivert source and LGPL compliance information must accompany redistribution.\r\n"))
	if e != nil {
		return nil, e
	}
	if e = z.Close(); e != nil {
		return nil, e
	}
	return buf.Bytes(), nil
}
func packOffline(exe, out string) error {
	zipData, e := makePayload(exe)
	if e != nil {
		return e
	}
	own, e := os.Executable()
	if e != nil {
		return e
	}
	if _, e = readOwnPayload(); e == nil {
		return errors.New("Bu EXE zaten gömülü motor içeriyor; tekrar paketlenmesi gerekmiyor")
	}
	src, e := os.Open(own)
	if e != nil {
		return e
	}
	defer src.Close()
	outtmp := out + ".tmp"
	dst, e := os.OpenFile(outtmp, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0700)
	if e != nil {
		return e
	}
	defer func() { _ = dst.Close(); _ = os.Remove(outtmp) }()
	if _, e = io.Copy(dst, src); e != nil {
		return e
	}
	if _, e = dst.Write(zipData); e != nil {
		return e
	}
	trailer := make([]byte, 48)
	copy(trailer[:8], payloadMagic)
	binary.LittleEndian.PutUint64(trailer[8:16], uint64(len(zipData)))
	sum := sha256.Sum256(zipData)
	copy(trailer[16:], sum[:])
	if _, e = dst.Write(trailer); e != nil {
		return e
	}
	if e = dst.Close(); e != nil {
		return e
	}
	return os.Rename(outtmp, out)
}
func (ap *app) handleVerifyMotor() {
	ap.work("Motorun dosya bütünlüğü kontrol ediliyor...", func() error {
		payload, e := readOwnPayload()
		if e != nil {
			return e
		}
		files, e := readEngineZIP(payload)
		if e != nil {
			return e
		}
		for name, b := range files {
			ap.enqueue(event{line: fmt.Sprintf("OK: %s (%d bytes)", name, len(b))})
		}
		ap.enqueue(event{line: "Motor ve sürücü EXE içinde. Ek bir indirme gerekmiyor.", state: "Motor doğrulandı"})
		return nil
	})
}

func (ap *app) handlePack() {
	ap.work("Çevrimdışı tek EXE hazırlanıyor. Yeni dosya arkadaşlarınızda indirme yapmaz.", func() error {
		exe := ap.path
		if exe == "" || checkEngineFiles(exe) != nil {
			exe = findGoodbyeDPI(ap.root)
		}
		if exe == "" || checkEngineFiles(exe) != nil {
			return errors.New("GoodbyeDPI, WinDivert.dll ve WinDivert64.sys aynı klasörde bulunmalı. Gerekirse bir kere 'ELLE SEÇ' ile mevcut motoru seçin")
		}
		home, _ := os.UserHomeDir()
		outdir := filepath.Join(home, "Desktop")
		if st, e := os.Stat(outdir); e != nil || !st.IsDir() {
			outdir = ap.root
		}
		out := filepath.Join(outdir, "DiscordBridge_TamPaket.exe")
		if _, e := os.Stat(out); e == nil {
			return fmt.Errorf("%s zaten var: yanlışlıkla üzerine yazılmaması için dosyayı taşıyın veya yeniden adlandırın", out)
		}
		if e := packOffline(exe, out); e != nil {
			return e
		}
		ap.enqueue(event{line: "HAZIR: " + out + " (GoodbyeDPI motoru paketin içinde)"})
		ap.enqueue(event{state: "Tek EXE hazır"})
		return nil
	})
}

func (ap *app) launchDPI(profile string, flags []string) error {
	enginePath := filepath.Join(ap.dependencyRoot(), "goodbyedpi.exe")
	if enginePath == "" {
		return errors.New("GoodbyeDPI bulunamadı")
	}
	if e := checkEngineFiles(enginePath); e != nil {
		return e
	}
	logPath := filepath.Join(ap.dataDir, "goodbyedpi.log")
	f, e := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if e != nil {
		return e
	}
	_, _ = io.WriteString(f, "\n[Discord Bridge v2] "+time.Now().Format(time.RFC3339)+" - "+profile+"\n")
	args := append(append([]string{}, flags...), "--blacklist", ap.domainsPath())
	cmd := exec.Command(enginePath, args...)
	cmd.Dir = filepath.Dir(enginePath)
	cmd.Stdout = f
	cmd.Stderr = f
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if e := cmd.Start(); e != nil {
		f.Close()
		return e
	}
	meta, _ := json.Marshal(processInfo{PID: cmd.Process.Pid, Path: enginePath, Profile: profile})
	if e := os.WriteFile(ap.processPath(), meta, 0600); e != nil {
		_ = cmd.Process.Kill()
		f.Close()
		return e
	}
	go func() {
		e := cmd.Wait()
		f.Close()
		if e != nil {
			ap.enqueue(event{line: "GoodbyeDPI kapandı: " + e.Error()})
		}
	}()
	time.Sleep(400 * time.Millisecond)
	if !processRunning(cmd.Process.Pid, enginePath) {
		return fmt.Errorf("Motor hemen kapandı; sürücü/güvenlik engeli olabilir (günlük: %s)", logPath)
	}
	return nil
}
func (ap *app) stopOwned() error {
	p, e := ap.readProcess()
	if e != nil {
		return nil
	}
	if processRunning(p.PID, p.Path) {
		script := fmt.Sprintf("$p=Get-CimInstance Win32_Process -Filter 'ProcessId = %d';if($p -and $p.ExecutablePath -ieq '%s'){Stop-Process -Id %d -Force -ErrorAction Stop; 'STOPPED'}else {throw 'Process identity mismatch'}", p.PID, escapePS(p.Path), p.PID)
		if _, e := powershell(script); e != nil {
			return e
		}
	}
	_ = os.Remove(ap.processPath())
	r := false
	ap.enqueue(event{running: &r})
	return nil
}

func checkHTTPS() error {
	// Prefer curl.exe provided with supported Windows versions: mirrors the
	// command line check confirmed successful on the original Superonline case.
	urls := []string{"https://updates.discord.com/", "https://discord.com/app"}
	curlWorked := true
	for _, raw := range urls {
		cmd := exec.Command("curl.exe", "-4", "-I", "-sS", "--connect-timeout", "5", "--max-time", "10", raw)
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		out, err := cmd.CombinedOutput()
		if err != nil || !bytes.Contains(out, []byte("HTTP/")) {
			curlWorked = false
			break
		}
	}
	if curlWorked {
		return nil
	}
	// If curl is missing or incompatible on a given Windows installation,
	// fall back to a native Go HTTPS probe.
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	tr := &http.Transport{DisableKeepAlives: true, ForceAttemptHTTP2: false,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return (&net.Dialer{Timeout: 5 * time.Second}).DialContext(ctx, "tcp4", addr)
		},
	}
	defer tr.CloseIdleConnections()
	cli := &http.Client{Timeout: 10 * time.Second, Transport: tr, CheckRedirect: func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }}
	for _, raw := range urls {
		req, err := http.NewRequestWithContext(ctx, "HEAD", raw, nil)
		if err != nil {
			return err
		}
		resp, err := cli.Do(req)
		if err != nil {
			return fmt.Errorf("%s: %w", raw, err)
		}
		resp.Body.Close()
		if resp.StatusCode == 0 {
			return fmt.Errorf("%s boş HTTP yanıtı", raw)
		}
	}
	return nil
}

func (ap *app) handleStop() {
	ap.work(ap.L("Yalnızca bu uygulamanın başlattığı GoodbyeDPI kapatılıyor...", "Stopping only the GoodbyeDPI process started by this app..."), func() error {
		if e := ap.stopOwned(); e != nil {
			return e
		}
		f := false
		ap.enqueue(event{line: ap.L("Bağlantı durduruldu. ExitLag etkilenmedi.", "Connection stopped. ExitLag unaffected."), state: "DPI durduruldu", connected: &f, activeProfile: "none"})
		return nil
	})
}
func readCurrentDNS() (dnsBackup, error) {
	var d dnsBackup
	script := `$ErrorActionPreference='Stop';
$all=@(Get-NetAdapter -Physical | Where-Object {$_.Status -eq 'Up' -and $_.InterfaceDescription -notmatch 'ExitLag'});
if($all.Count -eq 0){throw 'Aktif fiziksel Ethernet/Wi-Fi bağdaştırıcısı bulunamadı'};
$routes=@(Get-NetRoute -DestinationPrefix '0.0.0.0/0' -ErrorAction SilentlyContinue | Sort-Object RouteMetric);
$nic=$null;
foreach($r in $routes){$nic=$all | Where-Object {$_.ifIndex -eq $r.InterfaceIndex} | Select-Object -First 1; if($nic){break}};
if(-not $nic){if($all.Count -ne 1){throw 'Birden fazla aktif fiziksel ağ var. Tek bağlantıyla tekrar deneyin.'};$nic=$all[0]};
$guid='{'+$nic.InterfaceGuid.ToString().Trim('{}')+'}';
$v4key='HKLM:\SYSTEM\CurrentControlSet\Services\Tcpip\Parameters\Interfaces\'+$guid;
$v6key='HKLM:\SYSTEM\CurrentControlSet\Services\Tcpip6\Parameters\Interfaces\'+$guid;
$p4=(Get-ItemProperty -LiteralPath $v4key -Name NameServer -ErrorAction SilentlyContinue).NameServer;
$p6=(Get-ItemProperty -LiteralPath $v6key -Name NameServer -ErrorAction SilentlyContinue).NameServer;
$bind=(Get-NetAdapterBinding -InterfaceIndex $nic.ifIndex -ComponentID ms_tcpip6 -ErrorAction SilentlyContinue);
[pscustomobject]@{Index=[int]$nic.ifIndex;Alias=[string]$nic.Name;GUID=$guid;V4Manual=(-not [string]::IsNullOrWhiteSpace([string]$p4));V6Manual=(-not [string]::IsNullOrWhiteSpace([string]$p6));V4=@((Get-DnsClientServerAddress -InterfaceIndex $nic.ifIndex -AddressFamily IPv4).ServerAddresses | Where-Object {$_});V6=@((Get-DnsClientServerAddress -InterfaceIndex $nic.ifIndex -AddressFamily IPv6).ServerAddresses | Where-Object {$_});IPv6Enabled=[bool]$bind.Enabled} | ConvertTo-Json -Compress -Depth 5`
	result, err := powershell(script)
	if err != nil {
		return d, err
	}
	err = json.Unmarshal([]byte(result), &d)
	if err != nil {
		return d, fmt.Errorf("DNS bilgisi okunamadı: %s", result)
	}
	return d, nil
}
func (ap *app) handleDNS() {
	if !ap.confirm(ap.L("Cloudflare DNS uygula?", "Apply Cloudflare DNS?"), ap.L("Etkin fiziksel ağın DNS ayarı 1.1.1.1 / 1.0.0.1 ve IPv6 eşdeğerlerine çevrilecek.\n\nÖNCE mevcut ayarlar yedeklenecek.\nExitLag ve IP yönlendirme ayarları değişmeyecek.\n\nDevam edilsin mi?", "Your active physical network adapter will use Cloudflare IPv4 and IPv6 DNS.\n\nCurrent settings are backed up FIRST.\nExitLag and routing settings will not be changed.\n\nContinue?")) {
		return
	}
	ap.work(ap.L("Fiziksel ağ ayarları okunuyor...", "Reading physical network adapter settings..."), func() error {
		d, e := readCurrentDNS()
		if e != nil {
			return e
		}
		if old, e := os.ReadFile(ap.backupPath()); e == nil {
			var b dnsBackup
			if json.Unmarshal(old, &b) != nil || !strings.EqualFold(b.GUID, d.GUID) {
				return errors.New("Önceki DNS yedeği başka bir bağlantıya ait. Önce geri alın; yedek üzerine yazılmadı.")
			}
			ap.enqueue(event{line: "Mevcut DNS yedeği korunuyor."})
		} else {
			data, _ := json.MarshalIndent(d, "", "  ")
			if e := os.WriteFile(ap.backupPath(), data, 0600); e != nil {
				return e
			}
			ap.enqueue(event{line: "DNS ayarları yedeklendi: " + d.Alias})
		}
		v6 := ""
		if d.IPv6Enabled {
			v6 = ",'2606:4700:4700::1111','2606:4700:4700::1001'"
		}
		script := fmt.Sprintf("$ErrorActionPreference='Stop';Set-DnsClientServerAddress -InterfaceIndex %d -ServerAddresses @('1.1.1.1','1.0.0.1'%s);Clear-DnsClientCache; 'DNS_OK'", d.Index, v6)
		if _, e := powershell(script); e != nil {
			return e
		}
		ap.enqueue(event{line: ap.L("Cloudflare DNS uygulandı: ", "Cloudflare DNS applied: ") + d.Alias, state: "DNS ayarlandı", dns: "Cloudflare"})
		return nil
	})
}
func psArgs(v []string) string {
	q := make([]string, 0, len(v))
	for _, s := range v {
		q = append(q, "'"+escapePS(s)+"'")
	}
	return strings.Join(q, ",")
}
func (ap *app) handleRestore() {
	if !ap.confirm(ap.L("DNS ayarları geri alınsın mı?", "Restore DNS settings?"), ap.L("Yedeklenen orijinal DNS ayarlarını geri yükleyeceğiz.\nGoodbyeDPI ve ExitLag ayarları değişmeyecek.\n\nDevam edilsin mi?", "The backed-up original DNS settings will be restored.\nGoodbyeDPI and ExitLag will not be changed.\n\nContinue?")) {
		return
	}
	ap.work(ap.L("Eski DNS ayarları geri yükleniyor...", "Restoring original DNS settings..."), func() error {
		raw, e := os.ReadFile(ap.backupPath())
		if e != nil {
			return errors.New("Geri alınacak DNS yedeği bulunamadı")
		}
		var d dnsBackup
		if e = json.Unmarshal(raw, &d); e != nil {
			return e
		}
		check := fmt.Sprintf("$nic=Get-NetAdapter -InterfaceIndex %d -ErrorAction Stop; if(('{'+$nic.InterfaceGuid.ToString().Trim('{}')+'}') -ine '%s'){throw 'Ağ adaptörü değişmiş, otomatik geri alma iptal edildi.'}", d.Index, escapePS(d.GUID))
		lines := []string{check}
		restore := func(family string, manual bool, addr []string) {
			if !manual || len(addr) == 0 {
				lines = append(lines, fmt.Sprintf("& netsh interface %s set dnsservers name=%d source=dhcp;if($LASTEXITCODE -ne 0){throw 'DNS geri alınamadı: %s'}", family, d.Index, family))
				return
			}
			lines = append(lines, fmt.Sprintf("& netsh interface %s set dnsservers name=%d source=static address='%s' validate=no;if($LASTEXITCODE -ne 0){throw 'DNS geri alınamadı: %s'}", family, d.Index, escapePS(addr[0]), family))
			for i := 1; i < len(addr); i++ {
				lines = append(lines, fmt.Sprintf("& netsh interface %s add dnsservers name=%d address='%s' index=%d validate=no;if($LASTEXITCODE -ne 0){throw 'DNS eklenemedi: %s'}", family, d.Index, escapePS(addr[i]), i+1, family))
			}
		}
		restore("ipv4", d.V4Manual, d.V4)
		if d.IPv6Enabled {
			restore("ipv6", d.V6Manual, d.V6)
		}
		lines = append(lines, "Clear-DnsClientCache;'DNS_RESTORED'")
		if _, e := powershell(strings.Join(lines, ";")); e != nil {
			return e
		}
		_ = os.Remove(ap.backupPath())
		ap.enqueue(event{line: ap.L("Eski DNS ayarları geri alındı: ", "Original DNS restored: ") + d.Alias, state: "DNS geri alındı", dns: "Geri alındı"})
		return nil
	})
}
func (ap *app) handleTest() {
	ap.work(ap.L("Discord DNS ve HTTPS bağlantısı test ediliyor...", "Testing Discord DNS and HTTPS..."), func() error {
		ips, err := net.LookupIP("updates.discord.com")
		if err != nil {
			return fmt.Errorf("DNS başarısız: %w", err)
		}
		if len(ips) == 0 {
			return errors.New("DNS hiç IP döndürmedi")
		}
		ap.enqueue(event{line: "DNS → " + ips[0].String()})
		for _, ip := range ips {
			if ip.String() == "195.175.254.2" || strings.EqualFold(ip.String(), "2a01:358:4014:a00::3") {
				return errors.New("DNS engelleme adresi döndü. DNS ayarını kontrol edin.")
			}
		}
		client := &http.Client{Timeout: 12 * time.Second, Transport: &http.Transport{DisableKeepAlives: true, ForceAttemptHTTP2: false, DialContext: (&net.Dialer{Timeout: 6 * time.Second}).DialContext}}
		resp, err := client.Get("https://updates.discord.com/")
		if err != nil {
			return fmt.Errorf("HTTPS başarısız: %w", err)
		}
		resp.Body.Close()
		ap.enqueue(event{line: fmt.Sprintf(ap.L("Discord güncelleme sunucusu yanıt verdi: HTTP %d", "Discord update server returned: HTTP %d"), resp.StatusCode)})
		resp2, err := client.Get("https://discord.com/app")
		if err != nil {
			return fmt.Errorf("Ana Discord bağlantısı başarısız: %w", err)
		}
		resp2.Body.Close()
		ap.enqueue(event{line: fmt.Sprintf(ap.L("Discord ana site yanıt verdi: HTTP %d", "Discord main site returned: HTTP %d"), resp2.StatusCode), state: "Bağlantı başarılı"})
		return nil
	})
}

// Shared by the application and native Windows UI regression tests.
func (ap *app) createMainWindow() error {
	ap.initGraphics()
	ap.brushBG = call(procCreateSolidBrush, uintptr(bg))
	ap.brushCard = call(procCreateSolidBrush, uintptr(card))
	ap.brushEdit = call(procCreateSolidBrush, uintptr(inputBG))
	ap.brushSidebar = call(procCreateSolidBrush, uintptr(sidebar))
	ap.fTitle = createFont(-31, 700)
	ap.fHeading = createFont(-19, 650)
	ap.fText = createFont(-17, 500)
	ap.fSmall = createFont(-15, 450)
	ap.fBtn = createFont(-20, 650)
	ap.fHero = createFont(-28, 720)
	ap.fTiny = createFont(-12, 600)
	wc := wndclass{cbSize: uint32(unsafe.Sizeof(wndclass{})), wndProc: syscall.NewCallback(wndproc), className: ptr("DiscordBridgeWin32App"), background: 0}
	wc.instance = call(procGetModule, 0)
	wc.icon = call(procLoadIcon, wc.instance, 1)
	wc.iconSmall = wc.icon
	if call(procRegisterClass, uintptr(unsafe.Pointer(&wc))) == 0 {
		return errors.New("Windows UI initialization failed")
	}
	ap.trayIcon = wc.icon
	if ap.trayIcon == 0 {
		ap.trayIcon = call(procLoadIcon, 0, 32512) // Fallback: Windows application icon.
	}
	windowStyle := uint32(wsOverlappedWindow&^(0x00040000|0x00010000) | 0x02000000)
	frame := rect{0, 0, 1080, 710}
	call(procAdjustWindowRect, uintptr(unsafe.Pointer(&frame)), uintptr(windowStyle), 0, 0)
	work := rect{0, 0, 1920, 1080}
	call(procSystemParametersInfo, 48, 0, uintptr(unsafe.Pointer(&work)), 0)
	frameW, frameH := frame.right-frame.left-1080, frame.bottom-frame.top-710
	ap.scale = 1
	for _, fit := range []float64{float64(work.right-work.left-24-frameW) / 1080, float64(work.bottom-work.top-24-frameH) / 710} {
		if fit > 0 && fit < ap.scale {
			ap.scale = fit
		}
	}
	bounds := rect{0, 0, ap.px(1080), ap.px(710)}
	call(procAdjustWindowRect, uintptr(unsafe.Pointer(&bounds)), uintptr(windowStyle), 0, 0)
	width, height := bounds.right-bounds.left, bounds.bottom-bounds.top
	x, y := work.left+(work.right-work.left-width)/2, work.top+(work.bottom-work.top-height)/2
	ap.hwnd = call(procCreateWindow, 0, uintptr(unsafe.Pointer(wc.className)), uintptr(unsafe.Pointer(ptr("Discord Bridge 4.6"))), uintptr(windowStyle), uintptr(x), uintptr(y), uintptr(width), uintptr(height), 0, 0, wc.instance, 0)

	if ap.hwnd == 0 {
		return errors.New("Windows UI initialization failed")
	}
	// Windows 10/11: dark native title bar and rounded window corners where supported.
	immersiveDark := int32(1)
	darkAttr := uint32(20)
	call(procDwmSetWindowAttribute, ap.hwnd, uintptr(darkAttr), uintptr(unsafe.Pointer(&immersiveDark)), uintptr(unsafe.Sizeof(immersiveDark)))
	corner := int32(2)
	call(procDwmSetWindowAttribute, ap.hwnd, 33, uintptr(unsafe.Pointer(&corner)), uintptr(unsafe.Sizeof(corner)))
	ap.logEdit = createControl("EDIT", "", esReadOnly|esMultiline|esAutoVScroll|wsVScroll, 0, 0, 0, 0, 2002)
	call(procShowWindow, ap.logEdit, swHide)
	ap.button(idStart, "", 240, 282, 399, 54, primary, white)
	ap.button(idOpenDiscord, "", 655, 282, 399, 54, surface, white)
	positions := [][4]int32{{240, 418, 399, 64}, {655, 418, 399, 64}, {240, 496, 192, 44}, {447, 496, 192, 44}, {655, 496, 192, 44}, {862, 496, 192, 44}}
	for i, id := range profileButtonIDs {
		v := positions[i]
		ap.button(id, "", v[0], v[1], v[2], v[3], surface, white)
	}
	ap.button(idDNS, "", 15, 294, 180, 40, sidebar, white)
	ap.button(idRestore, "", 15, 340, 180, 40, sidebar, white)
	ap.button(idTest, "", 15, 386, 180, 40, sidebar, white)
	ap.button(idOpenLog, "", 15, 432, 180, 40, sidebar, white)
	ap.button(idPack, "", 15, 478, 180, 40, sidebar, white)
	ap.button(idLanguage, "", 19, 580, 172, 36, card, white)
	ap.refreshLabels()
	return nil
}

func main() {
	runtime.LockOSThread()
	// Restore already running window instead of starting another instance / UAC prompt.
	existing := call(procFindWindow, uintptr(unsafe.Pointer(ptr("DiscordBridgeWin32App"))), 0)
	if existing != 0 {
		call(procShowWindow, existing, swRestore)
		call(procSetForeground, existing)
		call(procPostMessage, existing, wmTrayRestore, 0, 0)
		return
	}
	// GoodbyeDPI and DNS changes require elevation. Ask once when launching the GUI.
	if call(procIsAdmin) == 0 {
		exe, err := os.Executable()
		if err == nil {
			result := call(procShellExecute, 0, uintptr(unsafe.Pointer(ptr("runas"))), uintptr(unsafe.Pointer(ptr(exe))), 0, 0, swShow)
			if result <= 32 {
				call(procMessageBox, 0, uintptr(unsafe.Pointer(ptr("Yönetici izni verilmedi. Ağ ayarları değiştirilmedi."))), uintptr(unsafe.Pointer(ptr("Discord Bridge"))), mbIconWarning)
			}
		}
		return
	}
	singleton := call(procCreateMutex, 0, 0, uintptr(unsafe.Pointer(ptr("Local\\DiscordBridge-Single-Instance-4"))))
	if singleton == 0 {
		return
	}
	if call(procGetLastError) == 183 {
		call(procCloseHandle, singleton)
		existing = call(procFindWindow, uintptr(unsafe.Pointer(ptr("DiscordBridgeWin32App"))), 0)
		if existing != 0 {
			call(procShowWindow, existing, swRestore)
			call(procSetForeground, existing)
			call(procPostMessage, existing, wmTrayRestore, 0, 0)
		}
		return
	}
	defer call(procCloseHandle, singleton)
	exe, err := os.Executable()
	if err != nil {
		return
	}
	a = &app{root: filepath.Dir(exe), dataDir: stateDir(), uiState: uiState{status: "Bekliyor", mode: "auto", selectedProfile: "special", language: "tr"}, buttons: make(map[int]uintptr), styles: make(map[uint32]style), enabled: make(map[int]bool), done: make(chan struct{})}
	a.loadConfig()

	if err := a.createMainWindow(); err != nil {
		a.alert("Discord Bridge", err.Error())
		return
	}
	// Register the icon immediately so the X button only hides the UI.
	if !a.addTray() {
		a.logLine("UYARI: Sistem tepsisine simge eklenemedi. Çarpı simgeye küçültmeyebilir.")
	}
	call(procShowWindow, a.hwnd, swShow)
	call(procUpdateWindow, a.hwnd)
	a.logLine("Discord Bridge 4.6: Superonline + ExitLag preserved (-5 --native-frag with Discord blacklist).")

	go a.monitorTraffic()
	// Do not start connection or Discord until the user clicks Start.
	if _, e := readOwnPayload(); e == nil {
		a.logLine("Embedded GoodbyeDPI engine ready; awaiting Start.")
	} else {
		a.logLine("WARNING: embedded engine cannot be verified.")
	}
	go func() {
		if a.isRunning() {
			r := true
			a.enqueue(event{running: &r, state: "DPI aktif", line: "Önceki GoodbyeDPI işlemi hâlâ açık."})
		}
	}()
	var m message
	for {
		if int32(call(procGetMessage, uintptr(unsafe.Pointer(&m)), 0, 0, 0)) <= 0 {
			break
		}
		if call(procIsDialogMessage, a.hwnd, uintptr(unsafe.Pointer(&m))) != 0 {
			continue
		}
		call(procTranslateMessage, uintptr(unsafe.Pointer(&m)))
		call(procDispatchMessage, uintptr(unsafe.Pointer(&m)))
	}
}
