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
// connections in the same capture. Each case also asserts that it exercised what it is there for
// (segment, record and hello counts), so a change in pasta or OpenSSL cannot make it pass vacuously.
//
// With -fingerprint PATH it also checks the android-capture harness's fingerprint.py, which keeps
// raw bytes (so these must match exactly), on TCP and on QUIC (decrypting the Initial packets).
//
// Needs podman with pasta and network access. Not part of the Go tests: run it after changing the
// extractor.
//

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

const (
	checkImage = "docker.io/library/python:3.12-slim" // OpenSSL >= 3.5: ML-KEM key shares and a QUIC client
	toolsImage = "localhost/capture-tools"            // fingerprint.py's QUIC path needs `cryptography`
	hostA      = "www.cloudflare.com"
	hostB      = "www.google.com"
)

// opensslConn is one `openssl s_client` connection: the host and extra options.
type opensslConn struct {
	host string
	opts []string
}

// checkCase is one ground-truth case: the connections to make, the index of the one under test,
// the JA4 fields its options imply (transport and version, SNI flag, ALPN), pasta's options, and
// what it must exercise: TCP segments and TLS records carrying the hello, ClientHellos sent.
type checkCase struct {
	name     string
	conns    []opensslConn
	idx      int
	ja4      [3]string
	pasta    []string
	segments int
	records  int
	hellos   int
}

func newCase(name string, conns []opensslConn, idx int, ja4 [3]string) checkCase {
	return checkCase{name: name, conns: conns, idx: idx, ja4: ja4, pasta: []string{"--mtu", "1280"},
		segments: 1, records: 1, hellos: 1}
}

func (c checkCase) with(f func(*checkCase)) checkCase { f(&c); return c }

func conn(host string, opts ...string) opensslConn { return opensslConn{host, opts} }

// checkCases: with MTU 1280, an ML-KEM hello (~1.5 KB) spans two TCP segments.
var checkCases = []checkCase{
	newCase("ipv6-mlkem-2-segments", []opensslConn{conn(hostA, "-6")}, 0, [3]string{"t13", "d", "00"}).
		with(func(c *checkCase) { c.segments = 2 }),
	newCase("ipv4-mlkem-2-segments", []opensslConn{conn(hostA, "-4")}, 0, [3]string{"t13", "d", "00"}).
		with(func(c *checkCase) { c.segments = 2 }),
	newCase("ipv4-mtu576-3-segments", []opensslConn{conn(hostA, "-4")}, 0, [3]string{"t13", "d", "00"}).
		with(func(c *checkCase) { c.pasta, c.segments = []string{"--mtu", "576", "-4"}, 3 }),
	newCase("tls12-only", []opensslConn{conn(hostA, "-4", "-tls1_2")}, 0, [3]string{"t12", "d", "00"}),
	newCase("alpn-h2-http11", []opensslConn{conn(hostA, "-4", "-alpn", "h2,http/1.1")}, 0, [3]string{"t13", "d", "h2"}).
		with(func(c *checkCase) { c.segments = 2 }),
	newCase("x25519-small", []opensslConn{conn(hostA, "-4", "-groups", "x25519")}, 0, [3]string{"t13", "d", "00"}),
	newCase("4-records-over-3-segments", []opensslConn{conn(hostA, "-4", "-max_send_frag", "512")}, 0,
		[3]string{"t13", "d", "00"}).
		with(func(c *checkCase) { c.pasta, c.segments, c.records = []string{"--mtu", "576", "-4"}, 3, 4 }),
	newCase("hello-retry-request", []opensslConn{conn(hostA, "-4", "-groups", "ffdhe2048:x25519")}, 0,
		[3]string{"t13", "d", "00"}).with(func(c *checkCase) { c.hellos = 2 }),
	newCase("decoy-other-sni-first", []opensslConn{conn(hostB, "-4"), conn(hostA, "-4", "-alpn", "http/1.1")}, 1,
		[3]string{"t13", "d", "h1"}).with(func(c *checkCase) { c.segments = 2 }),
	newCase("same-sni-twice-first-wins", []opensslConn{conn(hostA, "-4", "-alpn", "h2"), conn(hostA, "-4")}, 0,
		[3]string{"t13", "d", "h2"}).with(func(c *checkCase) { c.segments = 2 }),
}

// fingerprintOnlyCases: extract selects by SNI and reads TCP only.
var fingerprintOnlyCases = []checkCase{
	newCase("no-sni", []opensslConn{conn(hostA, "-4", "-noservername")}, 0, [3]string{"t13", "i", "00"}),
	newCase("quic-h3", []opensslConn{conn(hostA, "-quic", "-alpn", "h3")}, 0, [3]string{"q13", "d", "h3"}).
		with(func(c *checkCase) { c.segments, c.records = 0, 0 }),
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
	pcap := filepath.Join(work, "capture.pcap")
	var script []string
	for i, conn := range c.conns {
		servername := "-servername " + conn.host
		if slices.Contains(conn.opts, "-noservername") {
			servername = ""
		}
		script = append(script, fmt.Sprintf("openssl s_client -msg -connect %s:443 %s %s </dev/null >/w/msg%d 2>&1",
			conn.host, servername, strings.Join(conn.opts, " "), i))
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
		b, err := os.ReadFile(filepath.Join(work, fmt.Sprintf("msg%d", i)))
		if err != nil {
			return "", nil, err
		}
		msgs = append(msgs, string(b))
	}
	return pcap, msgs, nil
}

// helloLayout counts the TCP data segments and TLS records that carry the hello of the idx-th
// TCP connection to port 443, connections ordered by their SYNs. It does not use the extractor's
// reassembly: these connections have no loss or reordering, so segments are taken in capture order.
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
		if p.tcp.SYN {
			if _, ok := payloads[key]; !ok {
				payloads[key] = nil
				order = append(order, key)
			}
		} else if _, ok := payloads[key]; ok && len(p.tcp.Payload) > 0 {
			payloads[key] = append(payloads[key], p.tcp.Payload)
		}
	}
	if idx >= len(order) {
		return 0, 0, fmt.Errorf("no TCP connection %d to port 443", idx)
	}
	var data []byte
	i, got := 0, 0
	for _, s := range payloads[order[idx]] {
		data = append(data, s...)
		segments++
		for i+5 <= len(data) && data[i] == 22 && got < helloLen {
			n := int(data[i+3])<<8 | int(data[i+4])
			if i+5+n > len(data) {
				break
			}
			records, got, i = records+1, got+n, i+5+n
		}
		if got >= helloLen {
			break
		}
	}
	return segments, records, nil
}

func ja4Fields(ja4 string) [3]string {
	if len(ja4) < 10 {
		return [3]string{}
	}
	return [3]string{ja4[:3], ja4[3:4], ja4[8:10]}
}

// checkOne runs one case and returns its problems and notes.
func checkOne(c checkCase, work, fingerprint string, tcp bool) (problems, notes []string) {
	pcap, msgs, err := captureCase(work, c)
	if err != nil {
		return []string{err.Error()}, nil
	}
	conn := c.conns[c.idx]
	sent, err := opensslHellos(msgs[c.idx])
	if err != nil || len(sent) == 0 {
		return []string{fmt.Sprintf("OpenSSL printed no ClientHello (%v): %q", err, lastBytes(msgs[c.idx], 300))}, nil
	}
	truth := sent[0]
	notes = append(notes, fmt.Sprintf("%d B hello", len(truth)))
	if len(sent) < c.hellos {
		problems = append(problems, fmt.Sprintf("OpenSSL sent %d ClientHello(s), the case needs %d", len(sent), c.hellos))
	} else if c.hellos > 1 {
		notes = append(notes, fmt.Sprintf("%d hellos sent", len(sent)))
	}
	if tcp {
		segments, records, err := helloLayout(pcap, c.idx, len(truth))
		if err != nil {
			problems = append(problems, err.Error())
		}
		notes = append(notes, fmt.Sprintf("%d segment(s), %d record(s)", segments, records))
		if segments < c.segments || records < c.records {
			problems = append(problems, fmt.Sprintf("the hello took %d segment(s) and %d record(s); the case needs %d and %d",
				segments, records, c.segments, c.records))
		}
	}
	if tcp && !slices.Contains(conn.opts, "-noservername") {
		got, fields, err := findHello(pcap, conn.host)
		if err != nil {
			problems = append(problems, "extract failed: "+err.Error())
		} else {
			want, err := normalize(truth)
			if err != nil || !bytes.Equal(got, want) {
				problems = append(problems, fmt.Sprintf("extract's bytes differ from OpenSSL's (%d B vs %d B)", len(got), len(truth)))
			}
			if ja4Fields(fields.ja4()) != c.ja4 {
				problems = append(problems, fmt.Sprintf("extract JA4 %s, want fields %v", fields.ja4(), c.ja4))
			}
			notes = append(notes, "extract "+fields.ja4())
		}
	}
	if fingerprint != "" {
		problems, notes = checkFingerprint(c, work, fingerprint, tcp, truth, problems, notes)
	}
	return problems, notes
}

// checkFingerprint checks fingerprint.py's view of the case against OpenSSL's hello.
func checkFingerprint(c checkCase, work, fingerprint string, tcp bool, truth []byte, problems, notes []string) ([]string, []string) {
	dir, err := filepath.Abs(filepath.Dir(fingerprint))
	if err != nil {
		return append(problems, err.Error()), notes
	}
	out, err := exec.Command("podman", "run", "--rm", "-v", work+":/w:Z", "-v", dir+":/h:ro,Z", toolsImage, "python3", "-I",
		"/h/"+filepath.Base(fingerprint), "/w/capture.pcap", "--json", "--raw-dir", "/w/raw").Output() // #nosec G204 -- the user's own tool
	if err != nil {
		return append(problems, "fingerprint.py failed: "+err.Error()), notes
	}
	var result struct {
		Connections []struct {
			Transport string `json:"transport"`
			JA4       string `json:"ja4"`
			SNI       string `json:"sni"`
		} `json:"connections"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return append(problems, "fingerprint.py output: "+err.Error()), notes
	}
	transport := "quic"
	if tcp {
		transport = "tcp"
	}
	var rows []int // indexes, in time order, of the connections on this transport
	for n, x := range result.Connections {
		if x.Transport == transport {
			rows = append(rows, n)
		}
	}
	if len(rows) != len(c.conns) {
		return append(problems, fmt.Sprintf("fingerprint.py found %d %s handshakes, want %d", len(rows), transport, len(c.conns))), notes
	}
	n := rows[c.idx]
	row := result.Connections[n]
	var raw []byte
	if files, _ := filepath.Glob(filepath.Join(work, "raw", fmt.Sprintf("%03d-*", n))); len(files) > 0 {
		raw, _ = os.ReadFile(files[0])
	}
	if len(raw) >= 4 && len(raw) > 4+handshakeLength(raw) { // TLS 1.2: more follows the hello
		raw = raw[:4+handshakeLength(raw)]
	}
	if !bytes.Equal(raw, truth) {
		problems = append(problems, fmt.Sprintf("fingerprint.py raw bytes differ from OpenSSL's (%d B vs %d B)", len(raw), len(truth)))
	}
	if ja4Fields(row.JA4) != c.ja4 {
		problems = append(problems, fmt.Sprintf("fingerprint.py JA4 %s, want fields %v", row.JA4, c.ja4))
	}
	wantSNI := c.conns[c.idx].host
	if slices.Contains(c.conns[c.idx].opts, "-noservername") {
		wantSNI = ""
	}
	if row.SNI != wantSNI {
		problems = append(problems, fmt.Sprintf("fingerprint.py SNI %q", row.SNI))
	}
	return problems, append(notes, "fingerprint "+row.JA4)
}

// checkExtract runs every case and prints a line per case. keep, when set, keeps each case's
// capture and -msg output under that directory.
func checkExtract(fingerprint, keep string) error {
	type run struct {
		c   checkCase
		tcp bool
	}
	var runs []run
	for _, c := range checkCases {
		runs = append(runs, run{c, true})
	}
	if fingerprint != "" {
		for _, c := range fingerprintOnlyCases {
			runs = append(runs, run{c, c.segments > 0})
		}
	}
	failed := 0
	for _, r := range runs {
		var work string
		var err error
		if keep != "" {
			work = filepath.Join(keep, r.c.name)
			err = os.MkdirAll(work, 0o755)
		} else {
			work, err = os.MkdirTemp("", "check-extract-")
		}
		var problems, notes []string
		if err != nil {
			problems = []string{err.Error()}
		} else {
			problems, notes = checkOne(r.c, work, fingerprint, r.tcp)
		}
		if keep == "" && work != "" {
			_ = os.RemoveAll(work)
		}
		status := "ok  "
		if len(problems) > 0 {
			status = "FAIL"
			failed++
		}
		fmt.Printf("%s %-27s %s\n", status, r.c.name, strings.Join(notes, "; "))
		for _, p := range problems {
			fmt.Printf("     %s\n", p)
		}
	}
	fmt.Printf("%d/%d cases agree with OpenSSL\n", len(runs)-failed, len(runs))
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
