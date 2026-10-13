package main

//
// analyze: a capture's traffic shape, to compare an app's behaviour with the test's (see
// desktop_profile/README.md): DNS queries, and for each TCP connection its server, SNI and ALPN,
// timing, bytes each way, how it closed, and optionally its TLS records.
//

import (
	"cmp"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
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

// tlsRecords lists a stream's TLS records, up to the first byte that does not start one.
func tlsRecords(stream []byte, times []timedOffset, fromClient bool) []tlsRecord {
	var out []tlsRecord
	k := 0
	for i := 0; i+5 <= len(stream); {
		kind, ok := recordKinds[stream[i]]
		if !ok {
			break
		}
		for k+1 < len(times) && times[k+1].offset <= i {
			k++
		}
		length := int(stream[i+3])<<8 | int(stream[i+4])
		r := tlsRecord{fromClient: fromClient, kind: kind, length: length}
		if len(times) > 0 {
			r.time = times[k].time
		}
		out = append(out, r)
		i += 5 + length
	}
	return out
}

var dnsTypes = map[layers.DNSType]string{layers.DNSTypeA: "A", layers.DNSTypeAAAA: "AAAA", 65: "HTTPS"}

// analyze prints a capture's DNS queries and TCP connections (and, with records, their TLS records).
func analyze(capturePath string, records bool) error {
	c, err := readCapture(capturePath)
	if err != nil {
		return err
	}
	since := func(t time.Time) float64 { return t.Sub(c.first).Seconds() }
	fmt.Printf("%d packets over %.1f s\n", c.frames, since(c.last))

	fmt.Printf("\nDNS queries (time, name, type):\n")
	seen := map[string]bool{}
	for _, p := range c.packets {
		if p.udp == nil || p.udp.DstPort != 53 || len(p.udp.Payload) <= 12 {
			continue
		}
		var dns layers.DNS
		if dns.DecodeFromBytes(p.udp.Payload, gopacket.NilDecodeFeedback) != nil || len(dns.Questions) == 0 {
			continue
		}
		q := dns.Questions[0]
		typ, ok := dnsTypes[q.Type]
		if !ok {
			typ = strconv.Itoa(int(q.Type))
		}
		if key := string(q.Name) + " " + typ; !seen[key] {
			seen[key] = true
			fmt.Printf("  %7.2f  %s %s\n", since(p.time), q.Name, typ)
		}
	}

	fmt.Printf("\nTCP connections (start, end, server, SNI, ALPN, bytes up/down, close):\n")
	conns := tcpConnections(c.packets)
	slices.SortStableFunc(conns, func(a, b *tcpConnection) int { return a.first.Compare(b.first) })
	for _, conn := range conns {
		up, upTimes := conn.up.bytes()
		down, downTimes := conn.down.bytes()
		sni, alpn := "-", "-"
		if hs := clientHello(up); hs != nil {
			if f, err := parseHello(hs); err == nil {
				if f.sni != "" {
					sni = f.sni
				}
				if len(f.alpn) > 0 {
					alpn = strings.Join(f.alpn, ",")
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
		fmt.Printf("  %7.2f %7.2f  %s:%-5d %-28s %-14s %7d/%-8d %s\n", since(conn.first), since(conn.last),
			conn.server, conn.serverPort, sni, alpn, len(up), len(down), closed)
		if records && sni != "-" {
			recs := append(tlsRecords(up, upTimes, true), tlsRecords(down, downTimes, false)...)
			slices.SortFunc(recs, func(a, b tlsRecord) int { // by time, then direction, kind and length
				return cmp.Or(a.time.Compare(b.time), cmp.Compare(a.direction(), b.direction()),
					cmp.Compare(a.kind, b.kind), cmp.Compare(a.length, b.length))
			})
			for _, r := range recs {
				fmt.Printf("            %7.2f %s %-5s %d\n", since(r.time), r.direction(), r.kind, r.length)
			}
		}
	}
	return nil
}
