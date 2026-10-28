package main

type event struct {
	connected          *bool
	activeProfile      string
	attempt            string
	lastWorking        string
	trafficSample      bool
	trafficUnavailable bool
	trafficRX          float64
	trafficTX          float64
	line               string
	state              string
	busy               *bool
	running            *bool
	setPath            string
	dns                string
}
type uiState struct {
	mode, selectedProfile, lastWorkingProfile, activeProfile, attemptingProfile, language string
	connected                                                                             bool
	trafficRX, trafficTX                                                                  []float64
	trafficAvailable                                                                      bool
	running, busy                                                                         bool
	status, detail, dnsState                                                              string
}
type visualState struct {
	busy, running, connected           bool
	status, dns, active, attempt, last string
}

func (ap *uiState) visualState() visualState {
	return visualState{ap.busy, ap.running, ap.connected, ap.status, ap.dnsState, ap.activeProfile, ap.attemptingProfile, ap.lastWorkingProfile}
}

type redrawKind int

const (
	redrawNone redrawKind = iota
	redrawTraffic
	redrawContent
)

type stateChange struct {
	redraw     redrawKind
	saveConfig bool
}

// State reduction is independent of Win32. Only changed content requests a full repaint.
func (ap *uiState) applyEvents(arr []event) stateChange {
	before := ap.visualState()
	out := stateChange{}
	for _, e := range arr {
		if e.state != "" {
			ap.status = e.state
		}
		if e.busy != nil {
			ap.busy = *e.busy
		}
		if e.running != nil {
			ap.running = *e.running
		}
		if e.connected != nil {
			ap.connected = *e.connected
		}
		if e.dns != "" {
			ap.dnsState = e.dns
		}
		if e.activeProfile != "" {
			ap.activeProfile = e.activeProfile
		}
		if e.attempt != "" {
			ap.attemptingProfile = e.attempt
		}
		if e.lastWorking != "" && e.lastWorking != ap.lastWorkingProfile {
			ap.lastWorkingProfile = e.lastWorking
			out.saveConfig = true
		}
		if e.setPath != "" {
			out.saveConfig = true
		}
		if e.trafficSample {
			ap.trafficAvailable = true
			ap.trafficRX = append(ap.trafficRX, e.trafficRX)
			ap.trafficTX = append(ap.trafficTX, e.trafficTX)
			if len(ap.trafficRX) > 60 {
				ap.trafficRX = ap.trafficRX[len(ap.trafficRX)-60:]
			}
			if len(ap.trafficTX) > 60 {
				ap.trafficTX = ap.trafficTX[len(ap.trafficTX)-60:]
			}
			out.redraw = redrawTraffic
		}
		if e.trafficUnavailable && ap.trafficAvailable {
			ap.trafficAvailable = false
			out.redraw = redrawTraffic
		}
	}
	if before != ap.visualState() {
		out.redraw = redrawContent
	}
	return out
}
