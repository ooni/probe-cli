package main

//
// analyze: a capture's traffic shape, to compare an app's behaviour with the test's (see
// desktop_profile/README.md): DNS queries, and for each TCP connection its server, SNI and ALPN,
// timing, bytes each way, how it closed, and optionally its TLS records.
//

import (
	"cmp"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"golang.org/x/crypto/cryptobyte"
)

// tlsRecord is one TLS record's header, and when its first byte arrived.
type tlsRecord struct {
	time       time.Time
	fromClient bool
	kind       string
	length     int
}

// direction is "C>" for the client's records and "<S" for the server's.
func (r tlsRecord) direction() string {
	if r.fromClient {
		return "C>"
	}
	return "<S"
}

var recordKinds = map[byte]string{20: "CCS", 21: "alert", 22: "hs", 23: "app"}

// tlsRecords lists a stream's TLS records (the last may be cut short), up to the first bytes that
// do not start one.
func tlsRecords(stream []byte, times []timedOffset, fromClient bool) []tlsRecord {
	var out []tlsRecord
	k := 0
	for rest := cryptobyte.String(stream); ; {
		offset := len(stream) - len(rest)
		var contentType uint8
		var version, length uint16
		if !rest.ReadUint8(&contentType) || !rest.ReadUint16(&version) || !rest.ReadUint16(&length) ||
			recordKinds[contentType] == "" || version>>8 != 3 {
			return out
		}
		for k+1 < len(times) && times[k+1].offset <= offset {
			k++
		}
		r := tlsRecord{fromClient: fromClient, kind: recordKinds[contentType], length: int(length)}
		if len(times) > 0 {
			r.time = times[k].time
		}
		out = append(out, r)
		rest.Skip(min(int(length), len(rest)))
	}
}

// dnsQuestion is a DNS query's first question.
func dnsQuestion(payload []byte) (q layers.DNSQuestion, ok bool) {
	defer func() { // gopacket v1.1.19 panics on some truncated questions
		if recover() != nil {
			ok = false
		}
	}()
	var dns layers.DNS
	if dns.DecodeFromBytes(payload, gopacket.NilDecodeFeedback) != nil || len(dns.Questions) == 0 {
		return q, false
	}
	return dns.Questions[0], true
}

// printable is s, quoted if it has bytes a terminal could take as control sequences: captures are
// untrusted.
func printable(s string) string {
	for _, r := range s {
		if r < 0x20 || r > 0x7e {
			return strconv.QuoteToASCII(s)
		}
	}
	return s
}

var dnsTypes = map[layers.DNSType]string{layers.DNSTypeA: "A", layers.DNSTypeAAAA: "AAAA", 65: "HTTPS"}

// analyze prints a capture's DNS queries and TCP connections (and, with records, their TLS records).
func analyze(w io.Writer, capturePath string, records bool) error {
	c, err := readCapture(capturePath)
	if err != nil {
		return err
	}
	since := func(t time.Time) float64 { return t.Sub(c.first).Seconds() }
	fmt.Fprintf(w, "%d packets over %.1f s\n", c.frames, since(c.last))

	fmt.Fprintf(w, "\nDNS queries (time, name, type):\n")
	seen := map[string]bool{}
	for _, p := range c.packets {
		if p.udp == nil || p.udp.DstPort != 53 {
			continue
		}
		q, ok := dnsQuestion(p.udp.Payload)
		if !ok {
			continue
		}
		typ, ok := dnsTypes[q.Type]
		if !ok {
			typ = strconv.Itoa(int(q.Type))
		}
		if key := string(q.Name) + " " + typ; !seen[key] {
			seen[key] = true
			fmt.Fprintf(w, "  %7.2f  %s %s\n", since(p.time), printable(string(q.Name)), typ)
		}
	}

	fmt.Fprintf(w, "\nTCP connections (start, end, server, SNI, ALPN, bytes up/down, close):\n")
	conns := tcpConnections(c.packets)
	slices.SortStableFunc(conns, func(a, b *tcpConnection) int { return a.first.Compare(b.first) })
	for _, conn := range conns {
		up, upTimes := conn.up.bytes()
		down, downTimes := conn.down.bytes()
		sni, alpn := "-", "-"
		if hs := clientHello(up); hs != nil {
			if f, err := parseHello(hs); err == nil {
				if f.sni != "" {
					sni = printable(f.sni)
				}
				if len(f.alpn) > 0 {
					alpn = printable(strings.Join(f.alpn, ","))
				}
			}
		}
		closed := "open"
		if cl := conn.closed; cl != nil {
			how, by := "FIN", "server"
			if cl.reset {
				how = "RST"
			}
			if cl.byClient {
				by = "client"
			}
			closed = fmt.Sprintf("%s by %s at %.2f", how, by, since(cl.time))
		}
		fmt.Fprintf(w, "  %7.2f %7.2f  %s:%-5d %-28s %-14s %7d/%-8d %s\n", since(conn.first), since(conn.last),
			conn.server, conn.serverPort, sni, alpn, len(up), len(down), closed)
		if records && sni != "-" {
			recs := append(tlsRecords(up, upTimes, true), tlsRecords(down, downTimes, false)...)
			slices.SortFunc(recs, func(a, b tlsRecord) int { // by time, then direction, kind and length
				return cmp.Or(a.time.Compare(b.time), cmp.Compare(a.direction(), b.direction()),
					cmp.Compare(a.kind, b.kind), cmp.Compare(a.length, b.length))
			})
			for _, r := range recs {
				fmt.Fprintf(w, "            %7.2f %s %-5s %d\n", since(r.time), r.direction(), r.kind, r.length)
			}
		}
	}
	return nil
}
