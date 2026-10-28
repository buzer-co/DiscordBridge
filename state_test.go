package main

import (
	"testing"
	"unsafe"
)

func TestTrafficNeverRefreshesConnectionControls(t *testing.T) {
	s := uiState{connected: true, running: true, activeProfile: "special", status: "Bağlantı hazır"}
	before := s.visualState()
	for i := 0; i < 10000; i++ {
		out := s.applyEvents([]event{{trafficSample: true, trafficRX: float64(i), trafficTX: float64(i) / 2}})
		if out.redraw != redrawTraffic || out.saveConfig {
			t.Fatalf("sample %d touched controls/config: %+v", i, out)
		}
	}
	if s.visualState() != before {
		t.Fatal("traffic changed connection state")
	}
	if len(s.trafficRX) != 60 || len(s.trafficTX) != 60 || s.trafficRX[0] != 9940 || s.trafficRX[59] != 9999 {
		t.Fatal("traffic history is not bounded to the last 60 samples")
	}
}

func TestBurstsPreserveContentChanges(t *testing.T) {
	s := uiState{}
	yes := true
	out := s.applyEvents([]event{{busy: &yes}, {trafficSample: true, trafficRX: 20}, {connected: &yes, activeProfile: "special", lastWorking: "special"}, {trafficSample: true, trafficRX: 30}})
	if out.redraw != redrawContent || !out.saveConfig || !s.connected || s.activeProfile != "special" || s.lastWorkingProfile != "special" {
		t.Fatalf("burst lost a content transition: %+v %+v", s, out)
	}
}

func TestUnchangedEventsDoNotRequestRefresh(t *testing.T) {
	s := uiState{status: "Bekliyor", lastWorkingProfile: "special"}
	for _, events := range [][]event{nil, {{line: "log only"}}, {{state: "Bekliyor"}}, {{trafficUnavailable: true}}, {{lastWorking: "special"}}} {
		if out := s.applyEvents(events); out.redraw != redrawNone || out.saveConfig {
			t.Fatalf("unchanged event triggered work: %+v", out)
		}
	}
}

func TestNetworkDisconnectAndRecovery(t *testing.T) {
	s := uiState{}
	s.applyEvents([]event{{trafficSample: true, trafficRX: 12}})
	if out := s.applyEvents([]event{{trafficUnavailable: true}}); out.redraw != redrawTraffic || s.trafficAvailable {
		t.Fatal("disconnect was not represented")
	}
	if out := s.applyEvents([]event{{trafficUnavailable: true}}); out.redraw != redrawNone {
		t.Fatal("repeated disconnect caused repaint")
	}
	if out := s.applyEvents([]event{{trafficSample: true, trafficRX: 0}}); out.redraw != redrawTraffic || !s.trafficAvailable {
		t.Fatal("recovery not represented")
	}
}

func TestCounterResetAndAdapterChangesCannotSpike(t *testing.T) {
	before := map[uint64]byteCounter{1: {1000, 500}, 2: {2000, 600}, 3: {9000, 9000}}
	after := map[uint64]byteCounter{1: {1100, 530}, 2: {3, 4}, 4: {800000000, 900000000}}
	rx, tx := counterDelta(before, after)
	if rx != 100 || tx != 30 {
		t.Fatalf("adapter reset/removal/addition counted as traffic: %d %d", rx, tx)
	}
}

func TestWindowsNetworkTableABI(t *testing.T) {
	r := ifRow2{}
	if unsafe.Sizeof(r) != 1352 || unsafe.Offsetof(r.InOctets) != 1208 || unsafe.Offsetof(r.OutOctets) != 1280 || unsafe.Offsetof(r.Flags) != 1152 {
		t.Fatalf("MIB_IF_ROW2 ABI mismatch: size=%d in=%d out=%d flags=%d", unsafe.Sizeof(r), unsafe.Offsetof(r.InOctets), unsafe.Offsetof(r.OutOctets), unsafe.Offsetof(r.Flags))
	}
}
