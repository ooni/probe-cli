package main

//
// Offline tests for extract: known ClientHellos (the committed ones, with fresh random values)
// are written into synthetic captures of every layout the extractor claims to handle, and must
// come back byte for byte. They cover what real captures rarely produce on demand: reordered,
// retransmitted and overlapping segments, flows without their SYN, sequence numbers wrapping past
// 2^32, Ethernet padding, IPv6, and pcap and pcapng files.
// check-extract complements them with real OpenSSL connections.
//

import (
	"bytes"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcapgo"
)

var (
	client4, server4 = net.IPv4(192, 0, 2, 1).To4(), net.IPv4(198, 51, 100, 7).To4() // documentation addresses
	client6, server6 = net.ParseIP("2001:db8::1"), net.ParseIP("2001:db8::7")
)

// committed reads one of hellos/*.bin.
func committed(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "..", name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// withFreshRandoms is the hello as a client would send it: its random values filled in.
func withFreshRandoms(t *testing.T, hs []byte, rng *rand.Rand) []byte {
	t.Helper()
	spans, err := randomSpans(hs)
	if err != nil {
		t.Fatal(err)
	}
	out := append([]byte(nil), hs...)
	for _, sp := range spans {
		rng.Read(out[sp[0]:sp[1]])
	}
	if got, _ := normalize(out); !bytes.Equal(got, mustNormalize(t, hs)) {
		t.Fatal("normalize does not undo the fresh random values")
	}
	return out
}

func mustNormalize(t *testing.T, hs []byte) []byte {
	t.Helper()
	out, err := normalize(hs)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// withSNI is the hello with its server name replaced by one of the same length.
func withSNI(t *testing.T, hs []byte, sni string) []byte {
	t.Helper()
	f, err := parseHello(hs)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.sni) != len(sni) || bytes.Count(hs, []byte(f.sni)) != 1 {
		t.Fatalf("cannot replace %q with %q", f.sni, sni)
	}
	return bytes.Replace(hs, []byte(f.sni), []byte(sni), 1)
}

// records wraps hs in TLS handshake records, split at the given fragment sizes.
func records(hs []byte, sizes ...int) []byte {
	var out []byte
	for i, k := 0, 0; i < len(hs); k++ {
		n := len(hs) - i
		if k < len(sizes) {
			n = min(sizes[k], n)
		}
		out = append(out, 22, 3, 1, byte(n>>8), byte(n))
		out = append(out, hs[i:i+n]...)
		i += n
	}
	return out
}

// tcpPacket is one segment of a synthetic connection.
type tcpPacket struct {
	src, dst net.IP
	tcp      *layers.TCP
	payload  []byte
}

func tcpSegment(src, dst net.IP, sport, dport uint16, seq uint32, payload []byte) tcpPacket {
	return tcpPacket{src, dst, &layers.TCP{SrcPort: layers.TCPPort(sport), DstPort: layers.TCPPort(dport), Seq: seq,
		Ack: 1, ACK: true, PSH: len(payload) > 0, Window: 65535}, payload}
}

type flowOptions struct {
	cuts        []int
	order       []int
	noSYN       bool
	duplicate   [2]int // a retransmission of [start, end), with different boundaries
	v6          bool
	noServerRep bool
}

// flow is the client and server packets of one connection carrying stream, cut at the given
// offsets and sent in the given order (default: in order).
func flow(stream []byte, sport uint16, isn uint32, o flowOptions) []tcpPacket {
	c, s := client4, server4
	if o.v6 {
		c, s = client6, server6
	}
	edges := append(append([]int{0}, o.cuts...), len(stream))
	var data []tcpPacket
	for i := 0; i+1 < len(edges); i++ {
		data = append(data, tcpSegment(c, s, sport, 443, isn+1+uint32(edges[i]), stream[edges[i]:edges[i+1]]))
	}
	if d := o.duplicate; d != [2]int{} {
		data = append(data, tcpSegment(c, s, sport, 443, isn+1+uint32(d[0]), stream[d[0]:d[1]]))
	}
	if o.order != nil {
		reordered := make([]tcpPacket, len(o.order))
		for i, k := range o.order {
			reordered[i] = data[k]
		}
		data = reordered
	}
	var out []tcpPacket
	if !o.noSYN { // the final ACK has the hello's sequence number
		syn, synACK := tcpSegment(c, s, sport, 443, isn, nil), tcpSegment(s, c, 443, sport, 9000, nil)
		syn.tcp.SYN, syn.tcp.ACK, synACK.tcp.SYN = true, false, true
		out = append(out, syn, synACK, tcpSegment(c, s, sport, 443, isn+1, nil))
	}
	out = append(out, data...)
	if !o.noServerRep { // a server flight starting like a hello record: must not be taken for one
		out = append(out, tcpSegment(s, c, 443, sport, 9001, records([]byte{2, 0, 0, 4, 3, 3, 0, 0})))
	}
	return out
}

// transport is a TCP or UDP layer.
type transport interface {
	gopacket.SerializableLayer
	SetNetworkLayerForChecksum(gopacket.NetworkLayer) error
}

// ethernetFrame serializes a packet as pasta captures it: Ethernet (padded to its 60-byte
// minimum, as on a wire), IPv4 or IPv6, then TCP or UDP.
func ethernetFrame(t *testing.T, src, dst net.IP, l4 transport, payload []byte) []byte {
	t.Helper()
	eth := &layers.Ethernet{SrcMAC: net.HardwareAddr{2, 2, 2, 2, 2, 2}, DstMAC: net.HardwareAddr{4, 4, 4, 4, 4, 4},
		EthernetType: layers.EthernetTypeIPv4}
	var ip gopacket.SerializableLayer
	proto := layers.IPProtocolTCP
	if _, ok := l4.(*layers.UDP); ok {
		proto = layers.IPProtocolUDP
	}
	if src.To4() == nil {
		eth.EthernetType = layers.EthernetTypeIPv6
		ip = &layers.IPv6{Version: 6, HopLimit: 64, NextHeader: proto, SrcIP: src, DstIP: dst}
	} else {
		ip = &layers.IPv4{Version: 4, IHL: 5, TTL: 64, Protocol: proto, SrcIP: src, DstIP: dst}
	}
	if err := l4.SetNetworkLayerForChecksum(ip.(gopacket.NetworkLayer)); err != nil {
		t.Fatal(err)
	}
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{FixLengths: true, ComputeChecksums: true}
	if err := gopacket.SerializeLayers(buf, opts, eth, ip, l4, gopacket.Payload(payload)); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// captureWriter is what pcapgo's pcap and pcapng writers have in common.
type captureWriter interface {
	WritePacket(gopacket.CaptureInfo, []byte) error
}

// frames are a connection's packets as Ethernet frames.
func frames(t *testing.T, packets []tcpPacket) [][]byte {
	var out [][]byte
	for _, p := range packets {
		out = append(out, ethernetFrame(t, p.src, p.dst, p.tcp, p.payload))
	}
	return out
}

// writeCapture writes Ethernet frames one microsecond apart: as pcap, nanosecond pcap or pcapng.
func writeCapture(t *testing.T, path, format string, frames [][]byte) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var w captureWriter
	if format == "pcapng" {
		ng, err := pcapgo.NewNgWriter(f, layers.LinkTypeEthernet)
		if err != nil {
			t.Fatal(err)
		}
		defer ng.Flush() // before f.Close
		w = ng
	} else {
		pw := pcapgo.NewWriter(f)
		if format == "pcap-nanoseconds" {
			pw = pcapgo.NewWriterNanos(f)
		}
		if err := pw.WriteFileHeader(65535, layers.LinkTypeEthernet); err != nil {
			t.Fatal(err)
		}
		w = pw
	}
	for n, frame := range frames {
		ci := gopacket.CaptureInfo{Timestamp: time.Unix(1700000000, int64(n)*1000), CaptureLength: len(frame), Length: len(frame)}
		if err := w.WritePacket(ci, frame); err != nil {
			t.Fatal(err)
		}
	}
}

// fixture is one test's known hello, and the same hello with fresh random values.
type fixture struct {
	t     *testing.T
	dir   string
	rng   *rand.Rand
	hello []byte // android-okhttp-websocket.bin, SNI chat.signal.org
	sent  []byte
}

func newFixture(t *testing.T) *fixture {
	f := &fixture{t: t, dir: t.TempDir(), rng: rand.New(rand.NewSource(1))} // #nosec G404 -- test data
	f.hello = committed(t, "android-okhttp-websocket.bin")
	f.sent = withFreshRandoms(t, f.hello, f.rng)
	return f
}

// extractFrom writes packets as a capture and extracts sni's hello from it.
func (f *fixture) extractFrom(packets []tcpPacket, sni string, write func(path string)) []byte {
	f.t.Helper()
	path := filepath.Join(f.dir, "c.pcap")
	write(path)
	hs, _, err := findHello(path, sni)
	if err != nil {
		f.t.Fatal(err)
	}
	return hs
}

// roundtrip writes packets as a pcap and checks
// that chat.signal.org's hello comes back.
func (f *fixture) roundtrip(name string, packets []tcpPacket) {
	f.t.Helper()
	got := f.extractFrom(packets, "chat.signal.org", func(path string) { writeCapture(f.t, path, "pcap", frames(f.t, packets)) })
	if !bytes.Equal(got, f.hello) {
		f.t.Errorf("%s: extracted %d bytes, not the %d-byte hello", name, len(got), len(f.hello))
	}
}

func TestEveryCommittedHelloWhole(t *testing.T) {
	f := newFixture(t)
	for _, name := range []string{"libsignal.bin", "libsignal-reflector.bin", "android-okhttp-websocket.bin",
		"android-okhttp-rest.bin", "android-okhttp-front-g-gmail-rest.bin", "desktop-electron.bin"} {
		hs := committed(t, name)
		fields, err := parseHello(hs)
		if err != nil {
			t.Fatal(err)
		}
		packets := flow(records(withFreshRandoms(t, hs, f.rng)), 40000, 1000, flowOptions{})
		got := f.extractFrom(packets, fields.sni, func(path string) { writeCapture(t, path, "pcap", frames(t, packets)) })
		if !bytes.Equal(got, hs) {
			t.Errorf("%s: not extracted byte for byte", name)
		}
	}
}

func TestSegmentation(t *testing.T) {
	f := newFixture(t)
	stream := records(f.sent)
	for name, o := range map[string]flowOptions{
		"segmented and reordered":               {cuts: []int{100, 600, 1200}, order: []int{2, 0, 3, 1}},
		"retransmission with other boundaries":  {cuts: []int{500, 1000}, duplicate: [2]int{300, 900}, order: []int{0, 3, 1, 2}},
		"SYN not captured":                      {cuts: []int{700}, noSYN: true},
		"Ethernet padding on the handshake ACK": {}, // the bare ACK shares the hello's sequence number
	} {
		f.roundtrip(name, flow(stream, 40000, 1000, o))
	}
	f.roundtrip("fragmented into records across segments",
		flow(records(f.sent, 300, 300, 300), 40000, 1000, flowOptions{cuts: []int{298, 305, 1000}}))
	f.roundtrip("sequence numbers wrap inside the hello",
		flow(stream, 40000, 1<<32-600, flowOptions{cuts: []int{400, 800, 1200}}))
}

func TestIPv6(t *testing.T) {
	f := newFixture(t)
	f.roundtrip("IPv6", flow(records(f.sent), 40000, 1000, flowOptions{cuts: []int{700}, v6: true}))
}

func TestCaptureFormats(t *testing.T) {
	f := newFixture(t)
	packets := flow(records(f.sent), 40000, 1000, flowOptions{cuts: []int{700}})
	for _, format := range []string{"pcap-nanoseconds", "pcapng"} {
		got := f.extractFrom(packets, "chat.signal.org", func(path string) { writeCapture(t, path, format, frames(t, packets)) })
		if !bytes.Equal(got, f.hello) {
			t.Errorf("%s: not extracted byte for byte", format)
		}
	}
}

func TestRejectsOtherLinkTypes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "c.pcap")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := pcapgo.NewWriter(f).WriteFileHeader(65535, layers.LinkTypeLinuxSLL); err != nil {
		t.Fatal(err)
	}
	f.Close()
	if _, err := readCapture(path); err == nil || !strings.Contains(err.Error(), "not an Ethernet capture") {
		t.Errorf("err = %v", err)
	}
}

func TestPicksTheFirstConnectionWithTheSNI(t *testing.T) {
	f := newFixture(t)
	other := committed(t, "android-okhttp-rest.bin") // SNI cdn2.signal.org, as long as chat.signal.org
	decoy := flow(records(withFreshRandoms(t, other, f.rng)), 39999, 5000, flowOptions{})
	later := flow(records(withSNI(t, withFreshRandoms(t, other, f.rng), "chat.signal.org")), 40001, 7000, flowOptions{})
	packets := append(append(decoy, flow(records(f.sent), 40000, 1000, flowOptions{cuts: []int{700}})...), later...)
	f.roundtrip("first connection with the SNI", packets)
}

func TestFailsWhenTheSNIIsAbsent(t *testing.T) {
	f := newFixture(t)
	path, out := filepath.Join(f.dir, "c.pcap"), filepath.Join(f.dir, "out.bin")
	writeCapture(t, path, "pcap", frames(t, flow(records(f.sent), 40000, 1000, flowOptions{})))
	if err := run([]string{"extract", path, "example.org", out}); err == nil {
		t.Error("extract succeeded without a hello for the SNI")
	}
	if _, err := os.Stat(out); err == nil {
		t.Error("extract wrote a file")
	}
}

func TestNoSYNAndOutOfOrder(t *testing.T) {
	f := newFixture(t)
	f.roundtrip("first segment captured last", flow(records(f.sent), 40000, 1000,
		flowOptions{cuts: []int{400, 900}, order: []int{1, 2, 0}, noSYN: true}))
}

func TestPortReuseKeepsTheFirstConnection(t *testing.T) {
	f := newFixture(t)
	first := flow(records(f.sent), 40000, 1000, flowOptions{cuts: []int{700}})
	again := flow([]byte("GET / HTTP/1.1\r\n\r\n"), 40000, 50000, flowOptions{noServerRep: true})
	f.roundtrip("a later connection on the same ports", append(first, again...))
}

func TestParseHelloRejectsOtherMessages(t *testing.T) {
	hello := committed(t, "android-okhttp-websocket.bin")
	serverHello := append([]byte{2}, hello[1:]...)
	trailing := append(append([]byte(nil), hello...), 0)
	for name, hs := range map[string][]byte{"short": hello[:2], "ServerHello": serverHello,
		"truncated": hello[:len(hello)-1], "trailing byte": trailing} {
		if _, err := parseHello(hs); err == nil {
			t.Errorf("%s: parsed", name)
		}
	}
	if _, err := parseHello(hello); err != nil {
		t.Error(err)
	}
}

func TestSubcommandFlags(t *testing.T) {
	f := newFixture(t)
	path := filepath.Join(f.dir, "c.pcap")
	writeCapture(t, path, "pcap", frames(t, flow(records(f.sent), 40000, 1000, flowOptions{})))
	if err := run([]string{"analyze", "-records", path}); err != nil {
		t.Error(err)
	}
	for _, args := range [][]string{{"extract", "-write", path, "chat.signal.org", "out.bin"}, {"analyze", "-keep", "x", path}} {
		if err := run(args); err == nil {
			t.Errorf("%v: accepted another subcommand's flag", args)
		}
	}
}

// analyzeCapture writes frames as a capture and returns analyze's output.
func analyzeCapture(t *testing.T, frames [][]byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "c.pcap")
	writeCapture(t, path, "pcap", frames)
	var out bytes.Buffer
	if err := analyze(&out, path, true); err != nil {
		t.Fatal(err)
	}
	return out.String()
}

func dnsQuery(t *testing.T, payload []byte) []byte {
	udp := &layers.UDP{SrcPort: 40000, DstPort: 53}
	return ethernetFrame(t, client4, server4, udp, payload)
}

func TestAnalyzeDNS(t *testing.T) {
	query := []byte{0, 1, 1, 0, 0, 1, 0, 0, 0, 0, 0, 0, 4, 'c', 'h', 'a', 't', 6, 's', 'i', 'g', 'n', 'a', 'l', 3, 'o', 'r', 'g', 0, 0, 1, 0, 1}
	// a question cut short in its type: gopacket v1.1.19 reads past it, and panics when the frame
	// has no Ethernet padding to read into (18 bytes make a 60-byte frame)
	truncated := []byte{0, 1, 1, 0, 0, 1, 0, 0, 0, 0, 0, 0, 3, 'a', 'b', 'c', 0, 0}
	out := analyzeCapture(t, [][]byte{dnsQuery(t, truncated), dnsQuery(t, query)})
	if !strings.Contains(out, "chat.signal.org A\n") {
		t.Errorf("no query for chat.signal.org in\n%s", out)
	}
}

func TestAnalyzeQuotesCaptureText(t *testing.T) {
	f := newFixture(t)
	evil := "\x1b]0;x\x07" + strings.Repeat("a", len("chat.signal.org")-6) // as long as the SNI it replaces
	out := analyzeCapture(t, frames(t, flow(records(withSNI(t, f.sent, evil)), 40000, 1000, flowOptions{})))
	if strings.ContainsRune(out, 0x1b) || !strings.Contains(out, strconv.QuoteToASCII(evil)) {
		t.Errorf("SNI not quoted in\n%q", out)
	}
}

func TestTruncatedCapture(t *testing.T) {
	f := newFixture(t)
	path := filepath.Join(f.dir, "c.pcap")
	writeCapture(t, path, "pcap", frames(t, flow(records(f.sent), 40000, 1000, flowOptions{})))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data[:len(data)-10], 0o600); err != nil { // the last frame cut short
		t.Fatal(err)
	}
	got, _, err := findHello(path, "chat.signal.org")
	if err != nil || !bytes.Equal(got, f.hello) {
		t.Errorf("err %v; extracted %d bytes", err, len(got))
	}
}

func TestJA4CountsAndALPN(t *testing.T) {
	many := make([]uint16, 120)
	for i := range many {
		many[i] = uint16(0x1000 + i)
	}
	for _, c := range []struct {
		alpn string
		want string // the JA4's first part
	}{
		{"h2", "t13i9999h2"},
		{"h2-", "t13i99996d"}, // not alphanumeric at an end: the first and last hex digits ("68322d")
		{"\xab\xcd", "t13i9999ad"},
	} {
		f := &helloFields{versions: []uint16{0x0304}, ciphers: many, extensions: many, alpn: []string{c.alpn}}
		if got, _, _ := strings.Cut(f.ja4(), "_"); got != c.want {
			t.Errorf("ALPN %q: %s, want %s", c.alpn, got, c.want)
		}
	}
}
