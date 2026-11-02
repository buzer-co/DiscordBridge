package main

// DNS wire handling is deliberately independent of Windows and the packet driver.
import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

var discordDNSRoots = []string{"discord.com", "discord.gg", "discordapp.com", "discordapp.net", "discord.media"}

func discordDNSName(name string) bool {
	name = strings.ToLower(strings.TrimSuffix(name, "."))
	for _, root := range discordDNSRoots {
		if name == root || strings.HasSuffix(name, "."+root) {
			return true
		}
	}
	return false
}

type dnsQuestion struct {
	name     string
	end      int
	udpLimit int
}

// Only ordinary, one-question IN queries are intercepted. Unsupported packets
// (including compressed questions) pass through untouched rather than guessed.
func parseDNSQuestion(q []byte) (dnsQuestion, error) {
	bad := errors.New("unsupported DNS question")
	if len(q) < 17 || q[2]&0xf8 != 0 || binary.BigEndian.Uint16(q[4:6]) != 1 || binary.BigEndian.Uint16(q[6:8]) != 0 || binary.BigEndian.Uint16(q[8:10]) != 0 {
		return dnsQuestion{}, bad
	}
	i, total := 12, 0
	var labels []string
	for {
		if i >= len(q) {
			return dnsQuestion{}, bad
		}
		n := int(q[i])
		i++
		if n == 0 {
			break
		}
		if n > 63 || i+n > len(q) {
			return dnsQuestion{}, bad
		}
		label := q[i : i+n]
		for _, c := range label {
			if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_') {
				return dnsQuestion{}, bad
			}
		}
		labels = append(labels, string(label))
		i += n
		total += n + 1
		if total > 254 {
			return dnsQuestion{}, bad
		}
	}
	if len(labels) == 0 || i+4 > len(q) || binary.BigEndian.Uint16(q[i+2:i+4]) != 1 {
		return dnsQuestion{}, bad
	}
	i += 4
	d := dnsQuestion{name: strings.ToLower(strings.Join(labels, ".")), end: i, udpLimit: 512}
	ar := binary.BigEndian.Uint16(q[10:12])
	if ar == 0 && i == len(q) {
		return d, nil
	}
	// Optional single EDNS OPT record (root owner, type 41).
	if ar != 1 || i+11 > len(q) || q[i] != 0 || binary.BigEndian.Uint16(q[i+1:i+3]) != 41 || i+11+int(binary.BigEndian.Uint16(q[i+9:i+11])) != len(q) {
		return dnsQuestion{}, bad
	}
	d.udpLimit = int(binary.BigEndian.Uint16(q[i+3 : i+5]))
	if d.udpLimit < 512 {
		d.udpLimit = 512
	}
	if d.udpLimit > 1232 {
		d.udpLimit = 1232
	}
	return d, nil
}

func dnsErrorResponse(q []byte, code byte, truncated bool) []byte {
	d, err := parseDNSQuestion(q)
	if err != nil {
		return nil
	}
	r := append([]byte(nil), q[:d.end]...)
	r[2] = 0x80 | q[2]&1 // QR and original RD; never assert AD.
	if truncated {
		r[2] |= 2
	}
	r[3] = 0x80 | q[3]&0x10 | code // RA, CD and RCODE.
	clear(r[6:12])
	return r
}

func validateDNSResponse(q, r []byte) error {
	d, err := parseDNSQuestion(q)
	if err != nil {
		return err
	}
	if len(r) < d.end || r[2]&0xf8 != 0x80 || !bytes.Equal(q[:2], r[:2]) || binary.BigEndian.Uint16(r[4:6]) != 1 || !bytes.Equal(q[12:d.end], r[12:d.end]) {
		return errors.New("DNS response does not match question")
	}
	return nil
}

type dnsExchange func(context.Context, []byte) ([]byte, error)

type dohEndpoint struct {
	url    string
	client *http.Client
}

func newDoHEndpoint(host string, ips []string) dohEndpoint {
	tr := &http.Transport{
		Proxy: nil, ForceAttemptHTTP2: true, MaxConnsPerHost: 16, MaxIdleConnsPerHost: 4,
		TLSHandshakeTimeout: 2 * time.Second,
		TLSClientConfig:     &tls.Config{MinVersion: tls.VersionTLS12, ServerName: host},
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			// Bootstrap with literal addresses: poisoned system DNS cannot redirect
			// the resolver. Normal HTTPS hostname/certificate validation stays on.
			var last error
			for _, ip := range ips {
				conn, err := (&net.Dialer{Timeout: time.Second}).DialContext(ctx, "tcp4", net.JoinHostPort(ip, "443"))
				if err == nil {
					return conn, nil
				}
				last = err
			}
			return nil, last
		},
	}
	return dohEndpoint{"https://" + host + "/dns-query", &http.Client{Transport: tr, Timeout: 3 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
}

func (e dohEndpoint) exchange(ctx context.Context, q []byte) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", e.url, bytes.NewReader(q))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/dns-message")
	req.Header.Set("Accept", "application/dns-message")
	resp, err := e.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK || strings.ToLower(strings.TrimSpace(strings.Split(resp.Header.Get("Content-Type"), ";")[0])) != "application/dns-message" {
		return nil, fmt.Errorf("DoH HTTP %d or invalid content type", resp.StatusCode)
	}
	r, err := io.ReadAll(io.LimitReader(resp.Body, 65536))
	if err != nil || len(r) > 65535 {
		return nil, errors.New("invalid DoH response size")
	}
	if err = validateDNSResponse(q, r); err != nil {
		return nil, err
	}
	return r, nil
}

func newDiscordDoH() (dnsExchange, func()) {
	endpoints := []dohEndpoint{
		newDoHEndpoint("cloudflare-dns.com", []string{"1.1.1.1", "1.0.0.1"}),
		newDoHEndpoint("dns.google", []string{"8.8.8.8", "8.8.4.4"}),
	}
	return func(ctx context.Context, q []byte) ([]byte, error) {
		var last error
		for _, e := range endpoints {
			r, err := e.exchange(ctx, q)
			if err == nil && r[3]&15 != 2 {
				return r, nil
			}
			last = err
		}
		if last == nil {
			last = errors.New("DoH resolvers returned SERVFAIL")
		}
		return nil, last
	}, func() {
		for _, e := range endpoints {
			e.client.CloseIdleConnections()
		}
	}
}
