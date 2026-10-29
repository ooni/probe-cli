// Package browserhttptest checks that a client presents the fingerprint
// of the Chrome that [browserhttp] claims to be.
package browserhttptest

import (
	"bufio"
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"fmt"
	"io"
	"math/big"
	"net"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/andybalholm/brotli"
	"github.com/google/go-cmp/cmp"
	"github.com/klauspost/compress/zstd"
	"golang.org/x/crypto/cryptobyte"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/hpack"
)

// What a real Chrome 156 on Windows sends, as fingerprinted by tls.peet.ws
// and fp.impersonate.pro on 2026-10-25 (lexiforest/curl-impersonate#363),
// and the order of its navigation headers (curl-impersonate's chrome152
// signature, without cache-control, which Chrome sends only on reload).
const (
	ChromeJA4      = "t13d1517h2_8daaf6152771_cb7bf5808d99"
	ChromeAkamaiH2 = "1:65536;2:0;4:6291456;6:262144|15663105|0|m,a,s,p"
)

var ChromeH2Headers = []string{
	":method", ":authority", ":scheme", ":path",
	"sec-ch-ua", "sec-ch-ua-mobile", "sec-ch-ua-platform", "upgrade-insecure-requests",
	"user-agent", "accept", "sec-fetch-site", "sec-fetch-mode", "sec-fetch-user",
	"sec-fetch-dest", "accept-encoding", "accept-language", "priority",
}

var ChromeH1Headers = []string{
	"Host", "Connection",
	"sec-ch-ua", "sec-ch-ua-mobile", "sec-ch-ua-platform", "Upgrade-Insecure-Requests",
	"User-Agent", "Accept", "Sec-Fetch-Site", "Sec-Fetch-Mode", "Sec-Fetch-User",
	"Sec-Fetch-Dest", "Accept-Encoding", "Accept-Language",
}

// ChromeTrustAnchors are the trust anchor IDs Chrome 155 and 156 send, in
// their order (extension 0xca34).
var ChromeTrustAnchors = strings.Split("44947.2.1,44947.2.6,44947.2.13,44947.2.14,44947.2.15,"+
	"44947.2.18,44947.2.19,44947.2.20,52580.200109.1.7,52580.200109.1.8,52580.200109.1.9,"+
	"52580.200109.1.10,52580.200109.1.11,52580.200109.1.12,52580.200109.1.13,52580.200109.1.18,"+
	"52580.200109.1.19,11129.9.1,11129.9.4,11129.9.5,11129.9.6,11129.9.7,11129.9.8,11129.9.10,"+
	"11129.9.11,11129.9.12,11129.9.13,11129.9.15", ",")

// Body is the page the server sends, compressed.
const Body = "<html><title>hello</title></html>"

// Fingerprint is what the server saw of a client.
type Fingerprint struct {
	Hello    *ClientHello
	Akamai   string
	Headers  []string
	Priority http2.PriorityParam
}

// Check reports where fp differs from Chrome's, and whether the client
// decoded the body.
func Check(t testing.TB, proto string, fp *Fingerprint, body string) {
	t.Helper()
	if diff := cmp.Diff(ChromeJA4, fp.Hello.JA4()); diff != "" {
		t.Error("JA4", diff)
	}
	if len(fp.Hello.Sigalgs) == 0 || !isGREASE(fp.Hello.Sigalgs[0]) {
		t.Error("signature_algorithms doesn't start with GREASE")
	}
	if diff := cmp.Diff(ChromeTrustAnchors, fp.Hello.TrustAnchors); diff != "" {
		t.Error("trust anchors", diff)
	}
	if !slices.Contains(fp.Hello.Groups, 0x11ec) {
		t.Error("no X25519MLKEM768")
	}
	if proto == "h2" {
		if diff := cmp.Diff(ChromeAkamaiH2, fp.Akamai); diff != "" {
			t.Error("Akamai HTTP/2 fingerprint", diff)
		}
		if diff := cmp.Diff(ChromeH2Headers, fp.Headers); diff != "" {
			t.Error("HTTP/2 headers", diff)
		}
		if p := fp.Priority; p.StreamDep != 0 || !p.Exclusive || p.Weight != 255 {
			t.Errorf("HEADERS priority: got %+v, want weight 256, exclusive, on the root", p)
		}
	} else if diff := cmp.Diff(ChromeH1Headers, fp.Headers); diff != "" {
		t.Error("HTTP/1.1 headers", diff)
	}
	if body != Body {
		t.Errorf("decoded body: got %q", body)
	}
}

// Server answers one request per connection, negotiating Proto, and
// fingerprints the client. Its certificate is for www.example.com.
type Server struct {
	// CertPool trusts the server's certificate.
	CertPool *x509.CertPool

	cert    tls.Certificate
	ln      net.Listener
	proto   string
	results chan serverResult
}

type serverResult struct {
	fp  *Fingerprint
	err error
}

// NewServer starts a server on loopback that negotiates proto ("h2" or "http/1.1").
func NewServer(proto string) *Server {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}
	cert, pool := newCert()
	s := &Server{CertPool: pool, cert: cert, ln: ln, proto: proto, results: make(chan serverResult, 8)}
	go s.serve()
	return s
}

// Addr is the server's address.
func (s *Server) Addr() string {
	return s.ln.Addr().String()
}

// Close stops the server.
func (s *Server) Close() {
	_ = s.ln.Close()
}

// Fingerprint returns the fingerprint of the next client served.
func (s *Server) Fingerprint() (*Fingerprint, error) {
	select {
	case r := <-s.results:
		return r.fp, r.err
	case <-time.After(10 * time.Second):
		return nil, fmt.Errorf("no client")
	}
}

func (s *Server) serve() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			return
		}
		go func() {
			defer conn.Close()
			fp, err := s.fingerprint(conn)
			s.results <- serverResult{fp, err}
		}()
	}
}

// fingerprint answers one request on conn and records the ClientHello
// and the request's framing.
func (s *Server) fingerprint(conn net.Conn) (*Fingerprint, error) {
	proto := s.proto
	rec := &recordingConn{Conn: conn}
	tlsConn := tls.Server(rec, &tls.Config{
		Certificates: []tls.Certificate{s.cert},
		NextProtos:   []string{proto},
	})
	if err := tlsConn.Handshake(); err != nil {
		return nil, err
	}
	hello, err := parseClientHello(rec.read.Bytes())
	if err != nil {
		return nil, err
	}
	res := &Fingerprint{Hello: hello}
	if proto == "h2" {
		err = serveH2(tlsConn, res)
	} else {
		err = serveH1(tlsConn, res)
	}
	return res, err
}

func serveH2(conn net.Conn, res *Fingerprint) error {
	preface := make([]byte, len(http2.ClientPreface))
	if _, err := io.ReadFull(conn, preface); err != nil {
		return err
	}
	fr := http2.NewFramer(conn, conn)
	fr.ReadMetaHeaders = hpack.NewDecoder(4096, nil)
	var settings []string
	windowUpdate, priorityFrames := uint32(0), 0
	for {
		frame, err := fr.ReadFrame()
		if err != nil {
			return err
		}
		switch f := frame.(type) {
		case *http2.SettingsFrame:
			_ = f.ForeachSetting(func(s http2.Setting) error {
				settings = append(settings, fmt.Sprintf("%d:%d", s.ID, s.Val))
				return nil
			})
		case *http2.WindowUpdateFrame:
			if f.StreamID == 0 {
				windowUpdate = f.Increment
			}
		case *http2.PriorityFrame:
			priorityFrames++
		case *http2.MetaHeadersFrame:
			var pseudo []string
			for _, hf := range f.Fields {
				res.Headers = append(res.Headers, hf.Name)
				if hf.IsPseudo() {
					pseudo = append(pseudo, hf.Name[1:2])
				}
			}
			res.Priority = f.Priority
			res.Akamai = fmt.Sprintf("%s|%d|%d|%s", strings.Join(settings, ";"),
				windowUpdate, priorityFrames, strings.Join(pseudo, ","))

			var block bytes.Buffer
			enc := hpack.NewEncoder(&block)
			_ = enc.WriteField(hpack.HeaderField{Name: ":status", Value: "200"})
			_ = enc.WriteField(hpack.HeaderField{Name: "content-encoding", Value: "zstd"})
			if err := fr.WriteSettings(); err != nil {
				return err
			}
			if err := fr.WriteHeaders(http2.HeadersFrameParam{
				StreamID: 1, BlockFragment: block.Bytes(), EndHeaders: true}); err != nil {
				return err
			}
			return fr.WriteData(1, true, MustZstd(Body))
		}
	}
}

func serveH1(conn net.Conn, res *Fingerprint) error {
	r := bufio.NewReader(conn)
	if _, err := r.ReadString('\n'); err != nil { // request line
		return err
	}
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		name, _, _ := strings.Cut(line, ":")
		res.Headers = append(res.Headers, name)
	}
	body := MustBrotli(Body)
	_, err := fmt.Fprintf(conn, "HTTP/1.1 200 OK\r\nContent-Encoding: br\r\nContent-Length: %d\r\n\r\n%s", len(body), body)
	return err
}

// recordingConn records what the server reads.
type recordingConn struct {
	net.Conn
	read bytes.Buffer
}

func (c *recordingConn) Read(p []byte) (int, error) {
	n, err := c.Conn.Read(p)
	c.read.Write(p[:n])
	return n, err
}

// ClientHello is what fingerprints use from a ClientHello. Alpn and the
// lists keep the client's order and GREASE values.
type ClientHello struct {
	Ciphers      []uint16
	Extensions   []uint16
	Versions     []uint16
	Alpn         []string
	Sigalgs      []uint16
	Groups       []uint16
	TrustAnchors []string
}

func isGREASE(v uint16) bool {
	return v&0x0f0f == 0x0a0a && v>>8 == v&0xff
}

// parseClientHello parses the ClientHello at the start of a TLS stream.
func parseClientHello(data []byte) (*ClientHello, error) {
	s := cryptobyte.String(data)
	var record, hs, body cryptobyte.String
	var recordType uint8
	var recordVersion uint16
	var hsType uint8
	if !s.ReadUint8(&recordType) || recordType != 22 || !s.ReadUint16(&recordVersion) ||
		!s.ReadUint16LengthPrefixed(&record) || !record.ReadUint8(&hsType) || hsType != 1 ||
		!record.ReadUint24LengthPrefixed(&hs) {
		return nil, fmt.Errorf("not a ClientHello")
	}
	ch := &ClientHello{}
	var legacyVersion uint16
	var random, sessionID, compression, ciphers, exts cryptobyte.String
	if !hs.ReadUint16(&legacyVersion) || !hs.ReadBytes((*[]byte)(&random), 32) ||
		!hs.ReadUint8LengthPrefixed(&sessionID) || !hs.ReadUint16LengthPrefixed(&ciphers) ||
		!hs.ReadUint8LengthPrefixed(&compression) || !hs.ReadUint16LengthPrefixed(&exts) {
		return nil, fmt.Errorf("bad ClientHello")
	}
	for !ciphers.Empty() {
		var c uint16
		ciphers.ReadUint16(&c)
		ch.Ciphers = append(ch.Ciphers, c)
	}
	for !exts.Empty() {
		var typ uint16
		var data cryptobyte.String
		if !exts.ReadUint16(&typ) || !exts.ReadUint16LengthPrefixed(&data) {
			return nil, fmt.Errorf("bad extension")
		}
		ch.Extensions = append(ch.Extensions, typ)
		switch typ {
		case 43: // supported_versions
			data.ReadUint8LengthPrefixed(&body)
			ch.Versions = readUint16s(body)
		case 16: // ALPN
			data.ReadUint16LengthPrefixed(&body)
			for !body.Empty() {
				var p cryptobyte.String
				body.ReadUint8LengthPrefixed(&p)
				ch.Alpn = append(ch.Alpn, string(p))
			}
		case 13: // signature_algorithms
			data.ReadUint16LengthPrefixed(&body)
			ch.Sigalgs = readUint16s(body)
		case 10: // supported_groups
			data.ReadUint16LengthPrefixed(&body)
			ch.Groups = readUint16s(body)
		case 0xca34: // trust_anchors
			data.ReadUint16LengthPrefixed(&body)
			for !body.Empty() {
				var id cryptobyte.String
				body.ReadUint8LengthPrefixed(&id)
				ch.TrustAnchors = append(ch.TrustAnchors, relativeOID(id))
			}
		}
	}
	return ch, nil
}

func readUint16s(s cryptobyte.String) (out []uint16) {
	for !s.Empty() {
		var v uint16
		s.ReadUint16(&v)
		out = append(out, v)
	}
	return
}

// relativeOID decodes a trust anchor ID: base-128 arcs, as in an OID.
func relativeOID(b []byte) string {
	var arcs []string
	v := uint64(0)
	for _, c := range b {
		v = v<<7 | uint64(c&0x7f)
		if c&0x80 == 0 {
			arcs = append(arcs, strconv.FormatUint(v, 10))
			v = 0
		}
	}
	return strings.Join(arcs, ".")
}

// JA4 computes FoxIO's JA4 TLS client fingerprint (BSD-3-Clause spec).
func (ch *ClientHello) JA4() string {
	hexes := func(vs []uint16, skip ...uint16) (out []string) {
		for _, v := range vs {
			if !isGREASE(v) && !slices.Contains(skip, v) {
				out = append(out, fmt.Sprintf("%04x", v))
			}
		}
		return
	}
	hash := func(s string) string {
		sum := sha256.Sum256([]byte(s))
		return hex.EncodeToString(sum[:])[:12]
	}
	version := "00"
	for _, v := range ch.Versions {
		if !isGREASE(v) && v == tls.VersionTLS13 {
			version = "13"
		}
	}
	sni := "i"
	if slices.Contains(ch.Extensions, 0) {
		sni = "d"
	}
	alpn := "00"
	if len(ch.Alpn) > 0 && ch.Alpn[0] != "" {
		alpn = ch.Alpn[0][:1] + ch.Alpn[0][len(ch.Alpn[0])-1:]
	}
	ciphers := hexes(ch.Ciphers)
	exts := hexes(ch.Extensions)
	a := fmt.Sprintf("t%s%s%02d%02d%s", version, sni, min(len(ciphers), 99), min(len(exts), 99), alpn)
	slices.Sort(ciphers)
	sorted := hexes(ch.Extensions, 0, 16)
	slices.Sort(sorted)
	c := strings.Join(sorted, ",")
	if sigalgs := hexes(ch.Sigalgs); len(sigalgs) > 0 {
		c += "_" + strings.Join(sigalgs, ",")
	}
	return a + "_" + hash(strings.Join(ciphers, ",")) + "_" + hash(c)
}

// newCert returns a self-signed certificate for www.example.com and a pool
// that trusts it.
func newCert() (tls.Certificate, *x509.CertPool) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		panic(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		Subject:               pkix.Name{CommonName: "www.example.com"},
		DNSNames:              []string{"www.example.com"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		panic(err)
	}
	leaf, err := x509.ParseCertificate(der)
	if err != nil {
		panic(err)
	}
	pool := x509.NewCertPool()
	pool.AddCert(leaf)
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key, Leaf: leaf}, pool
}

// MustZstd compresses s with zstd.
func MustZstd(s string) []byte {
	enc, err := zstd.NewWriter(nil)
	if err != nil {
		panic(err)
	}
	return enc.EncodeAll([]byte(s), nil)
}

// MustBrotli compresses s with brotli.
func MustBrotli(s string) []byte {
	var buf bytes.Buffer
	w := brotli.NewWriter(&buf)
	if _, err := w.Write([]byte(s)); err != nil {
		panic(err)
	}
	if err := w.Close(); err != nil {
		panic(err)
	}
	return buf.Bytes()
}
