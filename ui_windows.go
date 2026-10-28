//go:build windows

package main

import (
	"fmt"
	"unsafe"
)

// The design stays at 1080x710; compact displays get one consistent scale.
func (ap *app) px(v int32) int32 {
	if ap.scale <= 0 || ap.scale == 1 {
		return v
	}
	return int32(float64(v)*ap.scale + 0.5)
}
func (ap *app) scaledRect(r rect) rect {
	return rect{ap.px(r.left), ap.px(r.top), ap.px(r.right), ap.px(r.bottom)}
}
func stretchCopy(dst uintptr, w, h int32, src uintptr, sw, sh int32) {
	if w == sw && h == sh {
		call(procBitBlt, dst, 0, 0, uintptr(w), uintptr(h), src, 0, 0, 0x00CC0020)
		return
	}
	call(procSetStretchBltMode, dst, 4)
	call(procSetBrushOrg, dst, 0, 0, 0)
	call(procStretchBlt, dst, 0, 0, uintptr(w), uintptr(h), src, 0, 0, uintptr(sw), uintptr(sh), 0x00CC0020)
}
func (ap *app) releaseBuffer() {
	if ap.bufferDC != 0 {
		call(procSelectObject, ap.bufferDC, ap.bufferOld)
		call(procDeleteObject, ap.bufferBitmap)
		call(procDeleteDC, ap.bufferDC)
	}
	ap.bufferDC, ap.bufferBitmap, ap.bufferOld = 0, 0, 0
}

func (ap *app) paintBuffered(h uintptr, dirty rect) {
	var r rect
	call(procGetClientRect, ap.hwnd, uintptr(unsafe.Pointer(&r)))
	actualW, actualH := r.right, r.bottom
	w, hgt := int32(1080), int32(710)
	if actualW <= 0 || actualH <= 0 {
		return
	}
	fresh := ap.bufferDC == 0 || ap.bufferW != w || ap.bufferH != hgt
	if fresh {
		ap.releaseBuffer()
		ap.bufferDC = call(procCreateCompatibleDC, h)
		ap.bufferBitmap = call(procCreateCompatibleBitmap, h, uintptr(w), uintptr(hgt))
		if ap.bufferDC == 0 || ap.bufferBitmap == 0 {
			ap.releaseBuffer()
			ap.paint(h)
			return
		}
		ap.bufferOld = call(procSelectObject, ap.bufferDC, ap.bufferBitmap)
		ap.bufferW, ap.bufferH = w, hgt
	}
	// The buffer retains the static artwork. Traffic paints only this panel.
	physicalNetwork := ap.scaledRect(networkRect)
	if !fresh && dirty.left >= physicalNetwork.left && dirty.top >= physicalNetwork.top && dirty.right <= physicalNetwork.right && dirty.bottom <= physicalNetwork.bottom {
		ap.paintNetwork(ap.bufferDC)
	} else {
		ap.paint(ap.bufferDC)
	}
	if dirty.right > dirty.left && dirty.bottom > dirty.top {
		stretchCopy(h, actualW, actualH, ap.bufferDC, w, hgt)
	}
}

func (ap *app) paint(hdc uintptr) {
	r := rect{0, 0, ap.bufferW, ap.bufferH}
	if r.right == 0 {
		r = rect{0, 0, 1080, 710}
	}
	call(procFillRect, hdc, uintptr(unsafe.Pointer(&r)), ap.brushBG)
	fillRound(hdc, rect{0, 0, 210, r.bottom}, sidebar, 0)
	softCircle(hdc, 22, 26, 46, primary, 255)
	if ap.trayIcon != 0 {
		call(procDrawIconEx, hdc, 30, 34, ap.trayIcon, 30, 30, 0, 0, 3)
	}
	drawText(hdc, "Discord", 80, 25, 112, 29, white, ap.fHeading, dtSingleLine)
	drawText(hdc, "BRIDGE", 81, 55, 114, 18, color(161, 166, 190), ap.fTiny, dtSingleLine)
	drawText(hdc, ap.L("BAĞLANTI ALANIN", "YOUR WORKSPACE"), 24, 111, 180, 20, muted, ap.fTiny, dtSingleLine)
	fillRound(hdc, rect{14, 145, 196, 191}, surface, 18)
	fillRound(hdc, rect{14, 157, 18, 179}, color(171, 179, 255), 4)
	softCircle(hdc, 30, 161, 13, primary, 255)
	drawText(hdc, ap.L("Genel Bakış", "Overview"), 55, 153, 135, 28, white, ap.fText, dtSingleLine|dtVCenter)
	drawText(hdc, ap.L("Her şey tek bir yerde.", "Everything in one place."), 24, 207, 176, 22, muted, ap.fSmall, dtSingleLine)
	drawText(hdc, ap.L("ARAÇLAR", "QUICK TOOLS"), 24, 266, 178, 20, muted, ap.fTiny, dtSingleLine)
	drawText(hdc, ap.L("DİL / LANGUAGE", "LANGUAGE / DİL"), 24, 551, 180, 20, muted, ap.fTiny, dtSingleLine)
	softCircle(hdc, 24, 649, 8, teal, 255)
	drawText(hdc, "ExitLag", 43, 638, 155, 23, grey, ap.fSmall, dtSingleLine)
	drawText(hdc, ap.L("Ayarların korunur", "Settings stay yours"), 43, 663, 155, 19, muted, ap.fTiny, dtSingleLine)

	drawText(hdc, ap.L("Bağlantı Merkezi", "Connection hub"), 240, 23, 610, 40, white, ap.fTitle, dtSingleLine)
	drawText(hdc, ap.L("Bağlan, Discord'u aç, keyfine bak.", "Connect, open Discord, make yourself at home."), 242, 68, 640, 24, grey, ap.fSmall, dtSingleLine)
	status, accent := ap.L("Beklemede", "Standby"), muted
	if ap.busy {
		status, accent = ap.L("İşlem sürüyor", "Working"), amber
	} else if ap.connected {
		status, accent = ap.L("Bağlı", "Connected"), teal
	} else if ap.running {
		status, accent = ap.L("Motor aktif", "Engine active"), amber
	}
	fillRound(hdc, rect{898, 30, 1054, 66}, card, 36)
	softCircle(hdc, 913, 44, 8, accent, 255)
	drawText(hdc, status, 932, 34, 111, 27, grey, ap.fSmall, dtSingleLine|dtVCenter)

	heroRect := rect{240, 108, 1054, 262}
	fillRound(hdc, heroRect, hero, 38)
	// Low contrast circles add depth while leaving the text area quiet.
	softCircle(hdc, 896, 121, 123, primary, 24)
	softCircle(hdc, 925, 150, 65, primary, 48)
	softCircle(hdc, 947, 172, 21, color(181, 188, 255), 150)
	label := ap.L("SEN HAZIRSAN, BİZ DE HAZIRIZ", "READY WHEN YOU ARE")
	heading := ap.L("Sohbete bir adım kaldı.", "One step closer to your people.")
	body := ap.L("Profilini seç veya gerisini otomatik bağlantıya bırak.", "Choose a profile, or let automatic discovery take care of it.")
	footer := ap.L("Discord, yalnızca sen istediğinde açılır.", "Discord opens only when you choose.")
	if ap.busy {
		label = ap.L("BAĞLANTI KONTROL EDİLİYOR", "CHECKING YOUR CONNECTION")
		heading = ap.L("Birazdan hazırız...", "Getting things ready...")
		body = ap.L("İşlem sürüyor. Ayrıntılar işlem günlüğüne kaydediliyor.", "Working. Details are saved in the activity log.")
		if ap.attemptingProfile != "" {
			body = ap.L("Denenen profil: ", "Trying profile: ") + ap.profileName(ap.attemptingProfile)
		}
		footer = ap.L("Çalışan profil bir sonraki açılışta hatırlanır.", "Your working profile is remembered for next time.")
	} else if ap.connected {
		label = ap.L("BAĞLANTIN HAZIR", "YOU'RE CONNECTED")
		heading = ap.L("Her şey hazır. Keyfine bak!", "You're all set. Enjoy!")
		body = ap.L("Aktif profil: ", "Active profile: ") + ap.profileName(ap.activeProfile)
		footer = ap.L("Discord'u aşağıdaki düğmeden açabilirsin.", "Open Discord using the button below.")
	} else if ap.status == "İşlem başarısız" {
		label = ap.L("BİR KONTROL GEREKİYOR", "LET'S CHECK THE CONNECTION")
		heading = ap.L("Bağlantıyı tekrar deneyelim.", "Let's try connecting again.")
		body = ap.L("Başka bir profil seçebilir, işlem günlüğünü inceleyebilirsin.", "Try another profile or check the activity log.")
	}
	drawText(hdc, label, 265, 127, 620, 22, color(181, 188, 228), ap.fTiny, dtSingleLine)
	drawText(hdc, heading, 264, 158, 640, 40, white, ap.fHero, dtSingleLine)
	drawText(hdc, body, 266, 204, 640, 24, grey, ap.fSmall, dtSingleLine)
	drawText(hdc, footer, 266, 235, 660, 20, color(162, 169, 198), ap.fTiny, dtSingleLine)

	drawText(hdc, ap.L("Sana uygun bağlantı", "Your connection, your way"), 241, 357, 600, 30, white, ap.fHeading, dtSingleLine)
	drawText(hdc, ap.L("Bir profil seç. Seçimin bir sonraki açılışta da burada.", "Pick a profile. Your choice will be here next time."), 242, 390, 730, 22, muted, ap.fSmall, dtSingleLine)
	ap.paintNetwork(hdc)
}

func (ap *app) paintNetwork(hdc uintptr) {
	// Clear the complete dirty rectangle first, including the rounded corners.
	call(procFillRect, hdc, uintptr(unsafe.Pointer(&networkRect)), ap.brushBG)
	fillRound(hdc, networkRect, card, 28)
	drawText(hdc, ap.L("Ağ aktivitesi", "Network activity"), 261, 579, 270, 26, white, ap.fHeading, dtSingleLine)
	if ap.trafficAvailable && len(ap.trafficRX) > 0 {
		i := len(ap.trafficRX) - 1
		drawText(hdc, fmt.Sprintf("↓ %.1f KB/s", ap.trafficRX[i]), 262, 619, 150, 26, color(108, 221, 185), ap.fText, dtSingleLine)
		drawText(hdc, fmt.Sprintf("↑ %.1f KB/s", ap.trafficTX[i]), 415, 619, 140, 26, color(171, 177, 255), ap.fText, dtSingleLine)
	} else {
		drawText(hdc, ap.L("Ağ verisi bekleniyor...", "Waiting for network data..."), 263, 619, 291, 25, grey, ap.fSmall, dtSingleLine)
	}
	drawText(hdc, ap.L("Tüm fiziksel ağ trafiği · son 60 sn", "All physical traffic · last 60 sec"), 263, 658, 360, 18, muted, ap.fTiny, dtSingleLine)
	drawNetworkChart(hdc, ap.trafficRX, ap.trafficTX, 578, 587, 451, 76)
}

func (ap *app) paintButton(di *drawItem) {
	actualW, actualH := di.rc.right-di.rc.left, di.rc.bottom-di.rc.top
	logical, ok := ap.controlRects[int(di.ctlID)]
	if !ok {
		logical = di.rc
	}
	w, h := logical.right-logical.left, logical.bottom-logical.top
	dc := call(procCreateCompatibleDC, di.hdc)
	bmp := call(procCreateCompatibleBitmap, di.hdc, uintptr(w), uintptr(h))
	if dc == 0 || bmp == 0 {
		if dc != 0 {
			call(procDeleteDC, dc)
		}
		if bmp != 0 {
			call(procDeleteObject, bmp)
		}
		ap.drawButton(di)
		return
	}
	old := call(procSelectObject, dc, bmp)
	copyItem := *di
	copyItem.hdc = dc
	copyItem.rc = rect{0, 0, w, h}
	ap.drawButton(&copyItem)
	stretchCopy(di.hdc, actualW, actualH, dc, w, h)
	call(procSelectObject, dc, old)
	call(procDeleteObject, bmp)
	call(procDeleteDC, dc)
}

func (ap *app) drawButton(di *drawItem) {
	s, ok := ap.styles[di.ctlID]
	if !ok {
		return
	}
	id := int(di.ctlID)
	r := di.rc
	w, h := r.right-r.left, r.bottom-r.top
	parentBrush := ap.brushBG
	if id != idStart && id != idOpenDiscord && (id < idProfileAuto || id > idProfile2) {
		parentBrush = ap.brushSidebar
	}
	call(procFillRect, di.hdc, uintptr(unsafe.Pointer(&r)), parentBrush)
	disabled := di.itemState&4 != 0
	hovered := ap.hoverID == id && !disabled
	pressed := di.itemState&diSelected != 0
	fg := white
	if disabled {
		fg = muted
	}
	if id == idStart || id == idOpenDiscord {
		c := s.fill
		if hovered {
			c = brighten(c, 13)
		}
		if pressed {
			c = brighten(c, -16)
		}
		if disabled {
			c = primarySoft
		}
		fillRound(di.hdc, r, c, 32)
		// Quiet circular icon, kept within the button's actual bounds.
		translucentRound(di.hdc, rect{13, 10, 47, h - 10}, white, 12, 22)
		icon := "→"
		if id == idStart {
			icon = "+"
			if ap.connected || ap.running {
				icon = "×"
			}
			if ap.busy {
				icon = "···"
			}
		}
		drawText(di.hdc, icon, 17, 5, 27, h-10, fg, ap.fHeading, dtCenter|dtVCenter|dtSingleLine)
		drawText(di.hdc, s.caption, 61, 0, w-78, h, fg, ap.fBtn, dtVCenter|dtSingleLine)
	} else if id >= idProfileAuto && id <= idProfile2 {
		idx := id - idProfileAuto
		selected := (ap.mode == "auto" && idx == 0) || (ap.mode == "manual" && idx > 0 && ap.selectedProfile == profileIDs[idx])
		c := surface
		if selected {
			c = selectedBG
		}
		if hovered {
			c = brighten(c, 9)
		}
		if pressed {
			c = brighten(c, -10)
		}
		fillRound(di.hdc, r, c, 28)
		if selected {
			outlineRound(di.hdc, rect{1, 1, w - 1, h - 1}, color(150, 162, 255), 14, 175)
		}
		dot := color(104, 108, 123)
		if selected {
			dot = color(157, 171, 255)
		}
		softCircle(di.hdc, 16, (h-14)/2, 14, dot, 255)
		if selected {
			softCircle(di.hdc, 20, (h-6)/2, 6, white, 255)
		}
		if idx < 2 {
			drawText(di.hdc, s.caption, 44, 9, w-57, 25, fg, ap.fText, dtSingleLine|dtVCenter)
			desc := ap.L("Çalışan profili bulur ve hatırlar", "Finds and remembers a working profile")
			if idx == 1 {
				desc = ap.L("Özel profil · öncelikli bağlantı", "Special profile · preferred connection")
			}
			drawText(di.hdc, desc, 44, 37, w-57, 20, color(170, 176, 197), ap.fTiny, dtSingleLine)
		} else {
			drawText(di.hdc, s.caption, 40, 0, w-47, h, fg, ap.fSmall, dtSingleLine|dtVCenter)
		}
	} else {
		c := sidebar
		if hovered {
			c = surface
		}
		if pressed {
			c = hoverSurface
		}
		if id == idLanguage {
			c = card
		}
		fillRound(di.hdc, r, c, 20)
		if id == idLanguage {
			drawText(di.hdc, s.caption, 4, 0, w-8, h, grey, ap.fSmall, dtSingleLine|dtCenter|dtVCenter)
		} else {
			fg = grey
			if hovered {
				fg = white
			}
			if disabled {
				fg = muted
			}
			icons := map[int]string{idDNS: "◎", idRestore: "↶", idTest: "~", idOpenLog: "≡", idPack: "◇"}
			drawText(di.hdc, icons[id], 10, 0, 23, h, muted, ap.fHeading, dtCenter|dtVCenter|dtSingleLine)
			drawText(di.hdc, s.caption, 42, 0, w-46, h, fg, ap.fSmall, dtSingleLine|dtVCenter)
		}
	}
	if di.itemState&0x10 != 0 {
		outlineRound(di.hdc, rect{3, 3, w - 3, h - 3}, color(200, 205, 255), 12, 220)
	}
}
