//go:build windows

package main

import (
	"fmt"
	"os"
	"strings"
	"unsafe"
)

type toolResult struct {
	state, message string
	lines          []string
}
type toolInfo struct{ title, subtitle, heading, description, hint, action, secondary string }

func isNavigation(id int) bool {
	switch id {
	case idOverview, idToolBack, idDNS, idRestore, idTest, idOpenLog, idPack:
		return true
	}
	return false
}

func (ap *app) toolInfo(id int) toolInfo {
	switch id {
	case idDNS:
		return toolInfo{
			ap.L("DNS merkezi", "DNS center"), ap.L("Mevcut ayarların ve Discord bağlantın.", "Your current settings and Discord connection."),
			ap.L("DNS ayarın sende kalsın.", "Keep your DNS settings."),
			ap.L("Yeni bağlantı yöntemi Discord'u kendi içinde çözer. Bağlanmak için sistem DNS'ini değiştirmen gerekmez.", "Discord uses the app's encrypted DNS bridge. Changing system DNS is not required to connect."),
			ap.L("Mevcut ayarları görüntüleyebilir veya istersen elle değiştirebilirsin.", "View current settings, or choose a manual change."),
			ap.L("Mevcut DNS'i göster", "Show current DNS"), ap.L("DNS'i elle değiştir", "Change DNS manually"),
		}
	case idRestore:
		return toolInfo{
			ap.L("DNS geri yükleme", "Restore DNS"), ap.L("Kaydettiğin ağ ayarlarına geri dön.", "Return to your saved network settings."),
			ap.L("Eski ayarların burada.", "Your previous settings, saved."),
			ap.L("Önceki bir elle DNS değişikliğinden kalan yedeği geri yükler. Yeni bağlantı yöntemi DNS yedeği oluşturmaz.", "Restore a backup from a previous manual DNS change. The new connection method does not create a DNS backup."),
			ap.L("Geri yükleme, onayından sonra başlar.", "Restoration starts only after your confirmation."), ap.L("Yedekten geri yükle", "Restore saved settings"), "",
		}
	case idTest:
		return toolInfo{
			ap.L("Bağlantı kontrolü", "Connection check"), ap.L("Discord'a giden yolu kontrol et.", "Check the path to Discord."),
			ap.L("Birlikte kontrol edelim.", "Let's check your connection."),
			ap.L("Discord adreslerini ve HTTPS erişimini test eder. Sonuçlar aşağıda görünür; bağlantıyı kendiliğinden başlatmaz.", "Test Discord name resolution and HTTPS access. Results appear below; this does not start the connection automatically."),
			ap.L("Şifreli Discord DNS'i için önce Genel Bakış'tan bağlantıyı başlat.", "Start the connection from Overview first to use encrypted Discord DNS."), ap.L("Testi başlat", "Run connection test"), "",
		}
	case idOpenLog:
		return toolInfo{
			ap.L("İşlem durumu", "Operation status"), ap.L("Son işlemin sonucu, tek bir yerde.", "The latest result in one place."),
			ap.L("Son işlem bilgisi.", "Your latest result."),
			ap.L("İşlem ayrıntıları bu ekran açıkken görünür. Dosyaya günlük yazılmaz; uygulama kapanınca sonuçlar silinir.", "Details are visible while this page is open. No log files are written; results disappear when the app closes."),
			ap.L("Sonraki açılış için yalnızca profil tercihin saklanır.", "Only your connection profile preference is saved for next time."), ap.L("Bağlantıyı kontrol et", "Check connection"), "",
		}

	case idPack:
		return toolInfo{
			ap.L("Motor kontrolü", "Engine check"), ap.L("Gömülü dosyaların durumunu kontrol et.", "Check the bundled engine files."),
			ap.L("Paketin eksiksiz mi?", "Is your package intact?"),
			ap.L("Programın içindeki GoodbyeDPI ve WinDivert dosyalarını denetler. İnternetten dosya indirmez ve bağlantıyı değiştirmez.", "Check the embedded GoodbyeDPI and WinDivert files. No downloads and no connection changes."),
			ap.L("Kontrol sonucu ve dosya ayrıntıları aşağıda görünür.", "The result and file details appear below."), ap.L("Motoru doğrula", "Verify engine"), "",
		}
	}
	return toolInfo{}
}

func (ap *app) toolResult(id int) *toolResult {
	if ap.toolResults == nil {
		ap.toolResults = map[int]*toolResult{}
	}
	if ap.toolResults[id] == nil {
		ap.toolResults[id] = &toolResult{state: "idle", message: ap.L("Hazır olduğunda yukarıdaki düğmeyle başlayabilirsin.", "Use the button above when you're ready.")}
	}
	return ap.toolResults[id]
}
func (ap *app) hasDNSBackup() bool {
	st, err := os.Stat(ap.backupPath())
	return err == nil && st.Mode().IsRegular()
}

func (ap *app) showPage(id int) {
	if id != 0 && !isNavigation(id) {
		return
	}
	if ap.page != id {
		if previous := ap.toolResults[ap.page]; previous != nil {
			previous.lines = nil
		}
	}
	ap.page = id
	for _, n := range append([]int{idStart, idOpenDiscord}, profileButtonIDs...) {
		show := swHide
		if id == 0 {
			show = swShow
		}
		call(procShowWindow, ap.buttons[n], uintptr(show))
	}
	for _, n := range []int{idToolAction, idToolBack} {
		show := swHide
		if id != 0 {
			show = swShow
		}
		call(procShowWindow, ap.buttons[n], uintptr(show))
	}
	showSecondary := swHide
	if id != 0 && ap.toolInfo(id).secondary != "" {
		showSecondary = swShow
	}
	call(procShowWindow, ap.buttons[idToolSecondary], uintptr(showSecondary))
	if id == 0 {
		call(procSetWindowText, ap.logEdit, uintptr(unsafe.Pointer(ptr(""))))
		call(procShowWindow, ap.logEdit, swHide)
	} else {
		r := ap.toolResult(id)
		if id == idRestore && !ap.hasDNSBackup() && r.state == "idle" {
			r.message = ap.L("Kaydedilmiş DNS yedeği bulunamadı. Geri yükleme gerekmiyor.", "No saved DNS backup was found. There is nothing to restore.")
		}
		// A native edit gives keyboard selection, scrolling and copy support.
		move := user32.NewProc("MoveWindow")
		move.Call(ap.logEdit, uintptr(ap.px(264)), uintptr(ap.px(431)), uintptr(ap.px(766)), uintptr(ap.px(173)), 0)
		call(procSendMessage, ap.logEdit, 0x00D3, 3, uintptr(12|12<<16)) // EM_SETMARGINS
		ap.renderToolLog()
		call(procShowWindow, ap.logEdit, swShow)
	}
	ap.refreshLabels()
	ap.updateButtons()
	ap.invalidate(nil)
	for _, n := range []int{idOverview, idDNS, idRestore, idTest, idOpenLog, idPack} {
		call(procInvalidateRect, ap.buttons[n], 0, 0)
	}
}

func (ap *app) renderToolLog() {
	if ap.logEdit == 0 || ap.page == 0 {
		return
	}
	lines := ap.toolResult(ap.page).lines
	if ap.page == idOpenLog && len(lines) == 0 && ap.detail != "" {
		lines = []string{ap.detail}
	}
	text := strings.Join(lines, "")
	if text == "" {
		text = ap.L("Henüz sonuç yok. İşlem tamamlandığında burada görünecek.", "No result yet. It will appear here when the action completes.")
	}
	call(procSetWindowText, ap.logEdit, uintptr(unsafe.Pointer(ptr(text))))
	call(procSendMessage, ap.logEdit, emSetSel, uintptr(^uint32(0)), uintptr(^uint32(0)))
	call(procSendMessage, ap.logEdit, 0x00B7, 0, 0) // EM_SCROLLCARET
}

func (ap *app) appendToolLog(line string) {
	ap.detail = strings.TrimSpace(line)
	if ap.activeTool != 0 && ap.page == ap.activeTool {
		r := ap.toolResult(ap.activeTool)
		r.lines = append(r.lines, line)
		if len(r.lines) > 40 {
			r.lines = r.lines[len(r.lines)-40:]
		}
	}
	if ap.page == idOpenLog || (ap.page != 0 && ap.page == ap.activeTool) {
		ap.renderToolLog()
	}
}

func (ap *app) runToolAction(secondary bool) {
	if ap.busy || ap.page == 0 {
		return
	}
	switch ap.page {
	case idDNS:
		if secondary {
			ap.handleDNS()
			if !ap.busy {
				r := ap.toolResult(idDNS)
				r.state = "idle"
				r.message = ap.L("İşlem iptal edildi. DNS ayarların değiştirilmedi.", "Cancelled. Your DNS settings were not changed.")
				ap.invalidate(nil)
			}
		} else {
			ap.work(ap.L("Mevcut DNS ayarları okunuyor...", "Reading current DNS settings..."), func() error {
				d, err := readCurrentDNS()
				if err != nil {
					return err
				}
				kind := ap.L("Otomatik", "Automatic")
				if d.V4Manual {
					kind = ap.L("Elle ayarlanmış", "Manual")
				}
				ap.enqueue(event{line: fmt.Sprintf("%s: %s\r\nIPv4: %s\r\nIPv6: %s\r\n%s", d.Alias, kind, strings.Join(d.V4, ", "), strings.Join(d.V6, ", "), ap.L("Bu işlem ayarları değiştirmedi.", "This check did not change settings."))})
				return nil
			})
		}
	case idRestore:
		if !ap.hasDNSBackup() {
			return
		}
		ap.handleRestore()
		if !ap.busy {
			r := ap.toolResult(idRestore)
			r.state = "idle"
			r.message = ap.L("Geri yükleme iptal edildi.", "Restoration cancelled.")
			ap.invalidate(nil)
		}
	case idTest:
		ap.handleTest()
	case idPack:
		ap.handleVerifyMotor()
	case idOpenLog:
		ap.handleTest()
	}
}

func (ap *app) paintToolPage(hdc uintptr) {
	info := ap.toolInfo(ap.page)
	drawText(hdc, info.title, 240, 23, 610, 40, white, ap.fTitle, dtSingleLine)
	drawText(hdc, info.subtitle, 242, 68, 780, 24, grey, ap.fSmall, dtSingleLine)
	fillRound(hdc, rect{240, 108, 1054, 256}, hero, 32)
	outlineRound(hdc, rect{241, 109, 1053, 255}, color(110, 120, 190), 16, 60)
	drawText(hdc, info.heading, 266, 128, 748, 38, white, ap.fHero, dtSingleLine)
	drawText(hdc, info.description, 267, 177, 738, 48, grey, ap.fSmall, 0x10)
	drawText(hdc, info.hint, 267, 229, 738, 20, color(175, 182, 216), ap.fTiny, dtSingleLine)
	fillRound(hdc, rect{240, 354, 1054, 628}, card, 32)
	r := ap.toolResult(ap.page)
	title, accent := ap.L("İşlem bekleniyor", "Ready when you are"), muted
	switch r.state {
	case "running":
		title, accent = ap.L("İşlem sürüyor…", "Working…"), amber
	case "success":
		title, accent = ap.L("İşlem tamamlandı", "Completed"), teal
	case "error":
		title, accent = ap.L("İşlem tamamlanamadı", "Could not complete"), red
	}
	if ap.page == idOpenLog && r.state == "idle" {
		title = ap.L("Son işlem sonucu", "Latest result")
	}
	softCircle(hdc, 266, 379, 10, accent, 255)
	drawText(hdc, title, 288, 367, 718, 30, white, ap.fHeading, dtSingleLine)
	drawText(hdc, r.message, 267, 405, 742, 22, grey, ap.fSmall, dtSingleLine)
	fillRound(hdc, rect{258, 427, 1036, 610}, inputBG, 16)
	drawText(hdc, ap.L("Günlük dosyası yok · Sonuçlar yalnızca bu oturumda görünür", "No log files · Results are visible only during this session"), 509, 654, 525, 22, muted, ap.fTiny, dtSingleLine)
}
