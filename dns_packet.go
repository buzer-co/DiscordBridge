package main

import "encoding/binary"

// A target contains a complete, unfragmented IPv4/IPv6 UDP DNS request.
// IPv6 extension headers and DNS over TCP are left untouched.
type dnsPacket struct {
	wire, query []byte
	udp         int
	question    dnsQuestion
}

func discordDNSPacket(wire []byte) (dnsPacket, bool) {
	p := dnsPacket{wire: wire}
	if len(wire) < 28 {
		return p, false
	}
	switch wire[0] >> 4 {
	case 4:
		p.udp = int(wire[0]&15) * 4
		if p.udp < 20 || len(wire) < p.udp+8 || wire[9] != 17 || binary.BigEndian.Uint16(wire[6:8])&0x3fff != 0 || int(binary.BigEndian.Uint16(wire[2:4])) != len(wire) {
			return p, false
		}
	case 6:
		p.udp = 40
		if len(wire) < 48 || wire[6] != 17 || int(binary.BigEndian.Uint16(wire[4:6]))+40 != len(wire) {
			return p, false
		}
	default:
		return p, false
	}
	u := wire[p.udp:]
	if binary.BigEndian.Uint16(u[2:4]) != 53 || int(binary.BigEndian.Uint16(u[4:6])) != len(u) {
		return p, false
	}
	p.query = u[8:]
	var err error
	p.question, err = parseDNSQuestion(p.query)
	return p, err == nil && discordDNSName(p.question.name)
}

func (p dnsPacket) response(payload []byte) []byte {
	if len(payload) > p.question.udpLimit {
		payload = dnsErrorResponse(p.query, 0, true)
	}
	r := make([]byte, p.udp+8+len(payload))
	copy(r, p.wire[:p.udp+8])
	copy(r[p.udp+8:], payload)
	if r[0]>>4 == 4 {
		copy(r[12:16], p.wire[16:20])
		copy(r[16:20], p.wire[12:16])
		binary.BigEndian.PutUint16(r[2:4], uint16(len(r)))
		r[8] = 64
		r[10], r[11] = 0, 0
	} else {
		copy(r[8:24], p.wire[24:40])
		copy(r[24:40], p.wire[8:24])
		binary.BigEndian.PutUint16(r[4:6], uint16(len(r)-40))
		r[7] = 64
	}
	u := r[p.udp:]
	copy(u[:2], p.wire[p.udp+2:p.udp+4])
	copy(u[2:4], p.wire[p.udp:p.udp+2])
	binary.BigEndian.PutUint16(u[4:6], uint16(len(u)))
	u[6], u[7] = 0, 0
	return r // WinDivertHelperCalcChecksums supplies IP/UDP checksums.
}
