package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func testDNSQuery(name string) []byte {
	q := []byte{0x42, 0x17, 1, 0, 0, 1, 0, 0, 0, 0, 0, 0}
	for _, s := range strings.Split(name, ".") {
		q = append(q, byte(len(s)))
		q = append(q, s...)
	}
	return append(q, 0, 0, 1, 0, 1)
}

func testDNSAnswer(q []byte) []byte {
	r := dnsErrorResponse(q, 0, false)
	r[7] = 1
	return append(r, 0xc0, 0x0c, 0, 1, 0, 1, 0, 0, 0, 30, 0, 4, 203, 0, 113, 7)
}

func testDNSWire(q []byte, ipv6 bool) []byte {
	h := 20
	if ipv6 {
		h = 40
	}
	w := make([]byte, h+8+len(q))
	copy(w[h+8:], q)
	if ipv6 {
		w[0], w[6], w[7] = 0x60, 17, 64
		binary.BigEndian.PutUint16(w[4:6], uint16(len(w)-40))
		w[8], w[23], w[24], w[39] = 0x20, 1, 0x20, 2
	} else {
		w[0], w[8], w[9] = 0x45, 64, 17
		binary.BigEndian.PutUint16(w[2:4], uint16(len(w)))
		copy(w[12:20], []byte{192, 0, 2, 1, 192, 0, 2, 53})
	}
	binary.BigEndian.PutUint16(w[h:h+2], 53123)
	binary.BigEndian.PutUint16(w[h+2:h+4], 53)
	binary.BigEndian.PutUint16(w[h+4:h+6], uint16(len(w)-h))
	return w
}

func TestDiscordDNSSelection(t *testing.T) {
	for _, s := range []string{"discord.com", "updates.discord.com", "US-East.discord.media", "cdn.discordapp.net.", "discord.gg"} {
		if !discordDNSName(s) {
			t.Errorf("missed %s", s)
		}
	}
	for _, s := range []string{"example.com", "notdiscord.com", "discord.com.evil.test", "discordapp.net.evil", "discord", ""} {
		if discordDNSName(s) {
			t.Errorf("intercepted unrelated name %q", s)
		}
	}
}

func TestDNSQuestionAndMalformedPackets(t *testing.T) {
	q := testDNSQuery("updates.discord.com")
	for n := 0; n < len(q); n++ {
		if _, err := parseDNSQuestion(q[:n]); err == nil {
			t.Fatalf("accepted truncated query %d", n)
		}
	}
	for _, mutate := range []func([]byte){
		func(b []byte) { b[2] |= 0x80 }, func(b []byte) { b[5] = 2 }, func(b []byte) { b[12] = 0xc0 },
		func(b []byte) { b[len(b)-1] = 3 }, func(b []byte) { b[6] = 1 },
	} {
		b := bytes.Clone(q)
		mutate(b)
		if _, e := parseDNSQuestion(b); e == nil {
			t.Fatal("accepted unsupported query")
		}
	}
	d, e := parseDNSQuestion(q)
	if e != nil || d.name != "updates.discord.com" || d.udpLimit != 512 {
		t.Fatal(d, e)
	}
	edns := bytes.Clone(q)
	edns[11] = 1
	edns = append(edns, 0, 0, 41, 0x10, 0, 0, 0, 0, 0, 0, 0)
	d, e = parseDNSQuestion(edns)
	if e != nil || d.udpLimit != 1232 {
		t.Fatal("EDNS limit", d, e)
	}
}

func TestDNSResponseValidation(t *testing.T) {
	q := testDNSQuery("discord.com")
	r := testDNSAnswer(q)
	if e := validateDNSResponse(q, r); e != nil {
		t.Fatal(e)
	}
	for _, offset := range []int{0, 2, 5, 13} {
		b := bytes.Clone(r)
		b[offset] ^= 0x80
		if validateDNSResponse(q, b) == nil {
			t.Fatalf("accepted mismatch at %d", offset)
		}
	}
	sf := dnsErrorResponse(q, 2, false)
	if sf[3]&15 != 2 || validateDNSResponse(q, sf) != nil {
		t.Fatal("invalid SERVFAIL")
	}
}

func TestDNSIPv4IPv6PacketReplyAndIsolation(t *testing.T) {
	for _, v6 := range []bool{false, true} {
		q := testDNSQuery("gateway.discord.gg")
		wire := testDNSWire(q, v6)
		p, ok := discordDNSPacket(wire)
		if !ok {
			t.Fatal("target not selected")
		}
		r := p.response(testDNSAnswer(q))
		h := p.udp
		if binary.BigEndian.Uint16(r[h:h+2]) != 53 || binary.BigEndian.Uint16(r[h+2:h+4]) != 53123 {
			t.Fatal("ports not reversed")
		}
		if v6 {
			if !bytes.Equal(r[8:24], wire[24:40]) || !bytes.Equal(r[24:40], wire[8:24]) {
				t.Fatal("IPv6 addresses not reversed")
			}
		} else {
			if !bytes.Equal(r[12:16], wire[16:20]) || !bytes.Equal(r[16:20], wire[12:16]) {
				t.Fatal("IPv4 addresses not reversed")
			}
		}
		if !bytes.Equal(r[h+8:], testDNSAnswer(q)) {
			t.Fatal("answer payload corrupted")
		}
		if _, ok = discordDNSPacket(testDNSWire(testDNSQuery("example.com"), v6)); ok {
			t.Fatal("unrelated DNS intercepted")
		}
		for n := 0; n < len(wire); n++ {
			if _, ok = discordDNSPacket(wire[:n]); ok {
				t.Fatal("truncated IP accepted")
			}
		}
		big := p.response(append(testDNSAnswer(q), make([]byte, 1300)...))
		if big[h+10]&2 == 0 || len(big) > h+8+512 {
			t.Fatal("oversized UDP answer not truncated")
		}
	}
	w := testDNSWire(testDNSQuery("discord.com"), false)
	w[6] = 0x20
	if _, ok := discordDNSPacket(w); ok {
		t.Fatal("fragment was interpreted as complete DNS")
	}
}

func TestDoHValidation(t *testing.T) {
	q := testDNSQuery("discord.com")
	for _, tc := range []struct {
		name   string
		status int
		kind   string
		body   []byte
		valid  bool
	}{
		{"valid", 200, "application/dns-message", testDNSAnswer(q), true},
		{"HTTP failure", 503, "application/dns-message", nil, false},
		{"HTML block page", 200, "text/html", []byte("blocked"), false},
		{"wrong question", 200, "application/dns-message", testDNSAnswer(testDNSQuery("example.com")), false},
		{"oversized", 200, "application/dns-message", make([]byte, 65536), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" || r.Header.Get("Content-Type") != "application/dns-message" {
					t.Error("wrong DoH request")
				}
				w.Header().Set("Content-Type", tc.kind)
				w.WriteHeader(tc.status)
				w.Write(tc.body)
			}))
			defer s.Close()
			e := dohEndpoint{s.URL, s.Client()}
			_, err := e.exchange(context.Background(), q)
			if (err == nil) != tc.valid {
				t.Fatal("validation mismatch", err)
			}
		})
	}
}

func FuzzDNSPacket(f *testing.F) {
	f.Add(testDNSWire(testDNSQuery("discord.com"), false))
	f.Add(testDNSWire(testDNSQuery("example.com"), true))
	f.Add([]byte{})
	f.Fuzz(func(t *testing.T, b []byte) {
		if p, ok := discordDNSPacket(b); ok {
			p.response(dnsErrorResponse(p.query, 2, false))
		}
	})
}
