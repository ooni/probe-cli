package main

//
// check-extract: the extractor against ground truth, real OpenSSL connections captured with pasta.
//
// For each case, `openssl s_client -msg` connects from a rootless container whose traffic pasta
// captures (--network pasta:--pcap). OpenSSL prints the exact ClientHello it sent; the extractor
// reads the capture. The two must agree byte for byte (after zeroing the random values, as
// extract does), and the JA4 fields that follow from the case's options (TLS version, SNI, ALPN)
// must hold. Cases cover IPv4 and IPv6, hellos split over two and three TCP segments, a hello
// fragmented over four TLS records, a HelloRetryRequest (the second hello must not win), and decoy
// connections in the same capture. Each case also checks that it exercised exactly what it is
// there for (segment, record and hello counts), so a change in pasta or OpenSSL cannot make it
// pass vacuously.
//
// Needs podman with pasta and network access. Not part of the Go tests: run it after changing the
// extractor.
//

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/cryptobyte"
)

const (
	checkImage = "docker.io/library/python:3.12-slim" // OpenSSL >= 3.5: ML-KEM key shares
	hostA      = "www.cloudflare.com"
	hostB      = "www.google.com"
)

// opensslConn is one `openssl s_client` connection: the host and extra options.
type opensslConn struct {
	host string
	opts []string
}

func conn(host string, opts ...string) opensslConn { return opensslConn{host, opts} }

// checkCase is one ground-truth case: the connections to make, the index of the one under test,
// the JA4 fields its options imply (transport and version, SNI flag, ALPN), pasta's options
// (default --mtu 1280), and what it must exercise: the TCP segments and TLS records carrying the
// hello, and the ClientHellos sent (each 1 when not given).
type checkCase struct {
	name                      string
	conns                     []opensslConn
	idx                       int
	ja4                       [3]string
	pasta                     []string
	segments, records, hellos int
}

// checkCases: with MTU 1280, an ML-KEM hello (~1.5 KB) spans two TCP segments.
var checkCases = []checkCase{
	{name: "ipv6-mlkem-2-segments", conns: []opensslConn{conn(hostA, "-6")}, ja4: [3]string{"t13", "d", "00"}, segments: 2},
	{name: "ipv4-mlkem-2-segments", conns: []opensslConn{conn(hostA, "-4")}, ja4: [3]string{"t13", "d", "00"}, segments: 2},
	{name: "ipv4-mtu576-3-segments", conns: []opensslConn{conn(hostA, "-4")}, ja4: [3]string{"t13", "d", "00"},
		pasta: []string{"--mtu", "576", "-4"}, segments: 3},
	{name: "tls12-only", conns: []opensslConn{conn(hostA, "-4", "-tls1_2")}, ja4: [3]string{"t12", "d", "00"}},
	{name: "alpn-h2-http11", conns: []opensslConn{conn(hostA, "-4", "-alpn", "h2,http/1.1")},
		ja4: [3]string{"t13", "d", "h2"}, segments: 2},
	{name: "x25519-small", conns: []opensslConn{conn(hostA, "-4", "-groups", "x25519")}, ja4: [3]string{"t13", "d", "00"}},
	{name: "4-records-over-3-segments", conns: []opensslConn{conn(hostA, "-4", "-max_send_frag", "512")},
		ja4: [3]string{"t13", "d", "00"}, pasta: []string{"--mtu", "576", "-4"}, segments: 3, records: 4},
	{name: "hello-retry-request", conns: []opensslConn{conn(hostA, "-4", "-groups", "ffdhe2048:x25519")},
		ja4: [3]string{"t13", "d", "00"}, hellos: 2},
	{name: "decoy-other-sni-first", conns: []opensslConn{conn(hostB, "-4"), conn(hostA, "-4", "-alpn", "http/1.1")},
		idx: 1, ja4: [3]string{"t13", "d", "h1"}, segments: 2},
	{name: "same-sni-twice-first-wins", conns: []opensslConn{conn(hostA, "-4", "-alpn", "h2"), conn(hostA, "-4")},
		ja4: [3]string{"t13", "d", "h2"}, segments: 2},
}

// withDefaults fills in the case's unset options.
func (c checkCase) withDefaults() checkCase {
	if c.pasta == nil {
		c.pasta = []string{"--mtu", "1280"}
	}
	c.segments, c.records, c.hellos = max(c.segments, 1), max(c.records, 1), max(c.hellos, 1)
	return c
}

var opensslHelloLine = regexp.MustCompile(`>>> .*Handshake \[length ([0-9a-f]+)\], ClientHello`)

// opensslHellos returns the ClientHello messages OpenSSL's -msg output says it sent, in order.
func opensslHellos(msg string) ([][]byte, error) {
	var out [][]byte
	lines := strings.Split(msg, "\n")
	for i, line := range lines {
		m := opensslHelloLine.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		var hexes []string
		for _, h := range lines[i+1:] {
			if !strings.HasPrefix(h, " ") {
				break
			}
			hexes = append(hexes, strings.ReplaceAll(strings.TrimSpace(h), " ", ""))
		}
		b, err := hex.DecodeString(strings.Join(hexes, ""))
		if err != nil {
			return nil, err
		}
		if want, _ := strconv.ParseInt(m[1], 16, 64); int64(len(b)) != want {
			return nil, fmt.Errorf("-msg length %s but %d bytes", m[1], len(b))
		}
		out = append(out, b)
	}
	return out, nil
}

// captureCase runs the case's connections one after another in one container, and returns the
// capture's path and each connection's -msg output.
func captureCase(work string, c checkCase) (string, []string, error) {
	if strings.ContainsAny(work, ",:") { // they separate podman's and pasta's options
		return "", nil, fmt.Errorf("%s: a working directory may not contain ',' or ':'", work)
	}
	pcap := filepath.Join(work, "capture.pcap")
	var script []string
	for i, conn := range c.conns {
		script = append(script, fmt.Sprintf("openssl s_client -msg -connect %s:443 -servername %s %s </dev/null >/w/msg%d 2>&1",
			conn.host, conn.host, strings.Join(conn.opts, " "), i))
	}
	network := strings.Join(append([]string{"pasta:--pcap", pcap}, c.pasta...), ",")
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "podman", "run", "--rm", "--network", network, "-v", work+":/w:Z", checkImage,
		"sh", "-c", strings.Join(script, "; ")) // #nosec G204 -- fixed cases
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", nil, fmt.Errorf("podman: %w: %s", err, lastBytes(string(out), 300))
	}
	var msgs []string
	for i := range c.conns {
		b, err := os.ReadFile(filepath.Join(work, fmt.Sprintf("msg%d", i))) // #nosec G304 -- our working directory
		if err != nil {
			return "", nil, err
		}
		msgs = append(msgs, string(b))
	}
	return pcap, msgs, nil
}

// helloLayout counts the TCP data segments and TLS records that carried the first helloLen bytes
// of handshake data on the idx-th TCP connection to port 443, connections ordered by their SYNs.
// It does not use the extractor's reassembly: these connections have no loss or reordering, so
// segments are taken in capture order.
func helloLayout(pcap string, idx, helloLen int) (segments, records int, err error) {
	c, err := readCapture(pcap)
	if err != nil {
		return 0, 0, err
	}
	type client struct {
		addr string
		port uint16
	}
	payloads := map[client][][]byte{}
	var order []client
	for _, p := range c.packets {
		if p.tcp == nil || p.tcp.DstPort != 443 {
			continue
		}
		key := client{p.src, uint16(p.tcp.SrcPort)}
		if _, seen := payloads[key]; p.tcp.SYN && !seen {
			payloads[key] = nil
			order = append(order, key)
		} else if seen && len(p.tcp.Payload) > 0 {
			payloads[key] = append(payloads[key], p.tcp.Payload)
		}
	}
	if idx >= len(order) {
		return 0, 0, fmt.Errorf("no TCP connection %d to port 443", idx)
	}
	var stream []byte
	for _, payload := range payloads[order[idx]] {
		stream = append(stream, payload...)
		segments++
		records, got := handshakeRecords(stream, helloLen)
		if got >= helloLen {
			return segments, records, nil
		}
	}
	return segments, 0, fmt.Errorf("the connection carried less than the %d-byte hello", helloLen)
}

// handshakeRecords counts the complete handshake records at the start of stream, up to the first
// that brings their data to want bytes, and returns how many bytes they carry.
func handshakeRecords(stream []byte, want int) (records, got int) {
	s := cryptobyte.String(stream)
	for got < want {
		var contentType uint8
		var version uint16
		var fragment cryptobyte.String
		if !s.ReadUint8(&contentType) || contentType != 22 || !s.ReadUint16(&version) ||
			!s.ReadUint16LengthPrefixed(&fragment) {
			break
		}
		records, got = records+1, got+len(fragment)
	}
	return records, got
}

func ja4Fields(ja4 string) [3]string {
	if len(ja4) < 10 {
		return [3]string{}
	}
	return [3]string{ja4[:3], ja4[3:4], ja4[8:10]}
}

// checkOne runs one case and returns its problems and notes.
func checkOne(c checkCase, work string) (problems, notes []string) {
	pcap, msgs, err := captureCase(work, c)
	if err != nil {
		return []string{err.Error()}, nil
	}
	sent, err := opensslHellos(msgs[c.idx])
	if err != nil || len(sent) == 0 {
		return []string{fmt.Sprintf("OpenSSL printed no ClientHello (%v): %q", err, lastBytes(msgs[c.idx], 300))}, nil
	}
	truth := sent[0]
	notes = append(notes, fmt.Sprintf("%d B hello", len(truth)))
	if len(sent) != c.hellos {
		problems = append(problems, fmt.Sprintf("OpenSSL sent %d ClientHello(s), the case needs %d", len(sent), c.hellos))
	} else if c.hellos > 1 {
		notes = append(notes, fmt.Sprintf("%d hellos sent", len(sent)))
	}

	segments, records, err := helloLayout(pcap, c.idx, len(truth))
	if err != nil {
		problems = append(problems, err.Error())
	}
	notes = append(notes, fmt.Sprintf("%d segment(s), %d record(s)", segments, records))
	if segments != c.segments || records != c.records {
		problems = append(problems, fmt.Sprintf("the hello took %d segment(s) and %d record(s); the case needs %d and %d",
			segments, records, c.segments, c.records))
	}

	got, fields, err := findHello(pcap, c.conns[c.idx].host)
	if err != nil {
		return append(problems, "extract failed: "+err.Error()), notes
	}
	if want, err := normalize(truth); err != nil {
		problems = append(problems, "OpenSSL's hello does not parse: "+err.Error())
	} else if !bytes.Equal(got, want) {
		problems = append(problems, fmt.Sprintf("extract's bytes differ from OpenSSL's (%d B vs %d B)", len(got), len(truth)))
	}
	if ja4Fields(fields.ja4()) != c.ja4 {
		problems = append(problems, fmt.Sprintf("extract JA4 %s, want fields %v", fields.ja4(), c.ja4))
	}
	return problems, append(notes, "extract "+fields.ja4())
}

// checkExtract runs every case and prints a line per case. keep, when set, keeps each case's
// capture and -msg output under that directory (replacing an earlier run's).
func checkExtract(keep string) error {
	failed := 0
	for _, c := range checkCases {
		c = c.withDefaults()
		var work string
		var err error
		if keep != "" {
			work = filepath.Join(keep, c.name)
			if err = os.RemoveAll(work); err == nil {
				err = os.MkdirAll(work, 0o750)
			}
		} else {
			work, err = os.MkdirTemp("", "check-extract-")
		}
		var problems, notes []string
		if err != nil {
			problems = []string{err.Error()}
		} else {
			problems, notes = checkOne(c, work)
		}
		if keep == "" && work != "" {
			_ = os.RemoveAll(work)
		}
		status := "ok  "
		if len(problems) > 0 {
			status = "FAIL"
			failed++
		}
		fmt.Printf("%s %-27s %s\n", status, c.name, strings.Join(notes, "; "))
		for _, p := range problems {
			fmt.Printf("     %s\n", p)
		}
	}
	fmt.Printf("%d/%d cases agree with OpenSSL\n", len(checkCases)-failed, len(checkCases))
	if failed > 0 {
		return fmt.Errorf("%d case(s) failed", failed)
	}
	return nil
}

func lastBytes(s string, n int) string {
	if len(s) > n {
		return s[len(s)-n:]
	}
	return s
}
