package main

//
// tlsprofile: a TLS front end's observable profile, to make the local Signal-Server match production.
//
// For each target (host[@address:port]) it records: TLS versions accepted; each TLS 1.3 and TLS 1.2
// cipher suite accepted, and which one the server picks from a full list (server preference);
// key-exchange groups accepted in TLS 1.3, and whether the server asks for another key share
// (HelloRetryRequest); the certificate chain; ALPN choices; session tickets; OCSP stapling; and
// the server's HTTP/2 SETTINGS and initial WINDOW_UPDATE.
//
// The TLS probes use `openssl s_client`, so that each one offers exactly what is asked. Its output
// is parsed as OpenSSL 3.5 prints it (testdata/ has samples). About 100 ordinary handshakes per
// target, paced. Run it where OpenSSL >= 3.5 is (for ML-KEM), e.g. via tlsprofile.sh.
//
//	TARGET: chat.signal.org  or  chat.signal.org@127.0.0.1:8443 (SNI chat.signal.org, address given)
//

import (
	"bytes"
	"cmp"
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"slices"
	"strings"
	"time"

	"golang.org/x/net/http2"
)

var (
	tls13Suites = []string{"TLS_AES_128_GCM_SHA256", "TLS_AES_256_GCM_SHA384", "TLS_CHACHA20_POLY1305_SHA256"}
	tls13Groups = []string{"X25519MLKEM768", "x25519", "P-256", "P-384", "P-521", "x448", "ffdhe2048", "ffdhe3072"}
)

const (
	probePause = 200 * time.Millisecond
	anyCipher  = "ALL:@SECLEVEL=0" // OpenSSL 3 refuses TLS 1.0/1.1 and weak suites at its default level
)

type tlsProfile struct {
	Target  string `json:"target"`
	SNI     string `json:"sni"`
	Address string `json:"address"`
	Error   string `json:"error,omitempty"` // the profile stops at the first probe that could not run

	Versions              map[string]bool        `json:"versions"`
	TLS13Suites           map[string]bool        `json:"tls13_suites"`
	TLS13ChoiceFromAll    string                 `json:"tls13_choice_from_all"`
	TLS12Suites           []string               `json:"tls12_suites"`
	TLS12ChoiceFromAll    []string               `json:"tls12_choice_from_all,omitempty"` // offered in two opposite orders
	TLS12ServerPreference *bool                  `json:"tls12_server_preference,omitempty"`
	TLS13Groups           map[string]groupResult `json:"tls13_groups"`
	X25519ShareFirst      groupResult            `json:"tls13_x25519_share_first"` // also offering P-256, X25519MLKEM768
	ALPN                  map[string]string      `json:"alpn"`                     // offered: chosen
	OCSPStapled           bool                   `json:"ocsp_stapled"`
	PeerSignature         string                 `json:"peer_signature"`
	ServerTempKey         string                 `json:"server_temp_key"`
	Chain                 []certInfo             `json:"chain"`
	TLS13SessionTickets   int                    `json:"tls13_session_tickets"`
	H2                    *h2Result              `json:"h2,omitempty"`
}

type groupResult struct {
	OK                bool   `json:"ok"`
	Group             string `json:"group,omitempty"`
	HelloRetryRequest bool   `json:"hello_retry_request"`
}

type certInfo struct {
	Subject    string   `json:"subject"`
	Issuer     string   `json:"issuer"`
	Key        string   `json:"key"`
	Signature  string   `json:"signature"`
	DNSNames   []string `json:"dns_names"`
	Extensions []string `json:"extensions"` // in order
}

type h2Result struct {
	ALPN         string            `json:"alpn"`
	Settings     map[string]uint32 `json:"settings"`
	WindowUpdate *uint32           `json:"window_update"` // the connection's first
}

func tlsProfiles(w io.Writer, targets []string) error {
	version, err := exec.Command("openssl", "version").Output()
	if err != nil {
		return fmt.Errorf("openssl: %w", err)
	}
	fmt.Fprintf(os.Stderr, "servertool: %s", version)
	var profiles []*tlsProfile
	for _, t := range targets {
		profiles = append(profiles, profile(t))
	}
	out, err := json.MarshalIndent(profiles, "", " ")
	if err != nil {
		return err
	}
	_, err = w.Write(append(out, '\n'))
	return err
}

// sessionResult is what one s_client run showed.
type sessionResult struct {
	out string
}

// field is the rest of the first line that starts with prefix (after spaces), or "".
func (r sessionResult) field(prefix string) string {
	for line := range strings.Lines(r.out) {
		if rest, ok := strings.CutPrefix(strings.TrimSpace(line), prefix); ok {
			return strings.TrimSpace(rest)
		}
	}
	return ""
}

func (r sessionResult) ok() bool { return r.cipher() != "" }

// cipher is the negotiated suite, from "New, TLSv1.3, Cipher is TLS_AES_256_GCM_SHA384".
func (r sessionResult) cipher() string {
	_, suite, _ := strings.Cut(r.field("New, "), "Cipher is ")
	if suite == "(NONE)" {
		return ""
	}
	return suite
}

// group is the key exchange: "Negotiated TLS1.3 group:" (hybrid groups only), else "Peer Temp Key:".
func (r sessionResult) group() string {
	if g := r.field("Negotiated TLS1.3 group:"); g != "" && g != "<NULL>" {
		return g
	}
	return r.field("Peer Temp Key:")
}

// helloRetryRequest needs -msg: the client sent a second ClientHello.
func (r sessionResult) helloRetryRequest() bool {
	return strings.Count(r.out, ", ClientHello\n") >= 2
}

// prober runs one target's s_client probes, paced, until one cannot run.
type prober struct {
	sni, addr string
	err       error
}

func (p *prober) run(args ...string) sessionResult { return p.runWithStdin(0, args...) }

// runWithStdin keeps s_client's input open for hold (it reads session tickets meanwhile).
func (p *prober) runWithStdin(hold time.Duration, args ...string) sessionResult {
	if p.err != nil {
		return sessionResult{}
	}
	time.Sleep(probePause)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	args = append([]string{"s_client", "-connect", p.addr, "-servername", p.sni, "-no_ign_eof"}, args...)
	cmd := exec.CommandContext(ctx, "openssl", args...) // #nosec G204 -- fixed program, the operator's targets
	stdin, w := io.Pipe()
	time.AfterFunc(hold, func() { w.Close() })
	var out bytes.Buffer
	cmd.Stdin, cmd.Stdout, cmd.Stderr = stdin, &out, &out
	_ = cmd.Run() // a failed handshake is a result
	if ctx.Err() != nil {
		p.err = fmt.Errorf("openssl %s: %w", strings.Join(args, " "), ctx.Err())
	}
	return sessionResult{out.String()}
}

func splitTarget(t string) (sni, addr string) {
	sni, addr, _ = strings.Cut(t, "@")
	if addr == "" {
		addr = net.JoinHostPort(sni, "443")
	}
	return sni, addr
}

func profile(target string) *tlsProfile {
	sni, addr := splitTarget(target)
	pr := &prober{sni: sni, addr: addr}
	p := &tlsProfile{Target: target, SNI: sni, Address: addr, Versions: map[string]bool{}, TLS13Suites: map[string]bool{},
		TLS12Suites: []string{}, TLS13Groups: map[string]groupResult{}, ALPN: map[string]string{}}

	for _, v := range []string{"tls1_3", "tls1_2", "tls1_1", "tls1"} {
		p.Versions[v] = pr.run("-"+v, "-cipher", anyCipher).ok()
	}
	for _, s := range tls13Suites {
		p.TLS13Suites[s] = pr.run("-tls1_3", "-ciphersuites", s).ok()
	}
	reversed := slices.Clone(tls13Suites)
	slices.Reverse(reversed)
	p.TLS13ChoiceFromAll = pr.run("-tls1_3", "-ciphersuites", strings.Join(reversed, ":")).cipher()

	suites, err := tls12Suites()
	if err != nil {
		p.Error = err.Error()
		return p
	}
	var accepted []string // OpenSSL names
	for _, s := range suites {
		if pr.run("-tls1_2", "-cipher", s.openssl+":@SECLEVEL=0").ok() {
			p.TLS12Suites = append(p.TLS12Suites, s.std)
			accepted = append(accepted, s.openssl)
		}
	}
	if len(accepted) > 1 { // the server prefers its own order if it picks the same from both
		a := pr.run("-tls1_2", "-cipher", strings.Join(accepted, ":")+":@SECLEVEL=0").cipher()
		slices.Reverse(accepted)
		b := pr.run("-tls1_2", "-cipher", strings.Join(accepted, ":")+":@SECLEVEL=0").cipher()
		p.TLS12ChoiceFromAll = []string{a, b}
		if a != "" && b != "" {
			preference := a == b
			p.TLS12ServerPreference = &preference
		}
	}

	for _, g := range tls13Groups { // offer only this group, with its key share
		r := pr.run("-tls1_3", "-groups", g, "-msg")
		p.TLS13Groups[g] = groupResult{OK: r.ok(), Group: r.group(), HelloRetryRequest: r.helloRetryRequest()}
	}
	r := pr.run("-tls1_3", "-groups", "x25519:P-256:X25519MLKEM768", "-msg") // a key share for x25519 only
	p.X25519ShareFirst = groupResult{OK: r.ok(), Group: r.group(), HelloRetryRequest: r.helloRetryRequest()}

	r = pr.run("-showcerts", "-status", "-alpn", "h2,http/1.1")
	p.ALPN["h2,http/1.1"] = r.field("ALPN protocol:")
	p.ALPN["http/1.1"] = pr.run("-alpn", "http/1.1").field("ALPN protocol:")
	p.ALPN["none"] = pr.run().field("ALPN protocol:")
	p.OCSPStapled = strings.Contains(r.out, "OCSP Response Status: successful")
	p.PeerSignature = strings.TrimSpace(r.field("Peer signature type:") + " " + r.field("Peer signing digest:"))
	p.ServerTempKey = r.group()
	p.Chain = certificateChain(r.out)
	p.TLS13SessionTickets = strings.Count(pr.runWithStdin(time.Second, "-tls1_3", "-sess_out", os.DevNull).out,
		"Post-Handshake New Session Ticket arrived")
	if pr.err != nil {
		p.Error = pr.err.Error()
		return p
	}
	if p.H2, err = h2Settings(sni, addr); err != nil {
		p.Error = err.Error()
	}
	return p
}

// cipherSuite is a TLS 1.2 suite's standard and OpenSSL names.
type cipherSuite struct{ std, openssl string }

// tls12Suites lists the TLS 1.2 suites this OpenSSL knows.
func tls12Suites() ([]cipherSuite, error) {
	out, err := exec.Command("openssl", "ciphers", "-s", "-tls1_2", "-stdname", "ALL:COMPLEMENTOFALL:@SECLEVEL=0").Output()
	return parseCipherList(string(out)), err
}

// parseCipherList reads `openssl ciphers -stdname` lines, such as
// "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256   - ECDHE-RSA-AES128-GCM-SHA256 TLSv1.2 Kx=ECDH ...",
// leaving out the TLS 1.3 suites.
func parseCipherList(out string) []cipherSuite {
	var suites []cipherSuite
	for line := range strings.Lines(out) {
		std, rest, ok := strings.Cut(line, " - ")
		std, fields := strings.TrimSpace(std), strings.Fields(rest)
		if ok && len(fields) > 0 && !slices.Contains(tls13Suites, std) {
			suites = append(suites, cipherSuite{std, fields[0]})
		}
	}
	return suites
}

// certificateChain describes the certificates in s_client's -showcerts output.
func certificateChain(out string) []certInfo {
	chain := []certInfo{}
	rest := []byte(out)
	for {
		var block *pem.Block
		if block, rest = pem.Decode(rest); block == nil {
			return chain
		}
		if cert, err := x509.ParseCertificate(block.Bytes); err == nil {
			chain = append(chain, describe(cert))
		}
	}
}

var extensionNames = map[string]string{
	"2.5.29.14": "subjectKeyIdentifier", "2.5.29.15": "keyUsage", "2.5.29.17": "subjectAltName",
	"2.5.29.19": "basicConstraints", "2.5.29.31": "cRLDistributionPoints", "2.5.29.32": "certificatePolicies",
	"2.5.29.35": "authorityKeyIdentifier", "2.5.29.37": "extKeyUsage", "1.3.6.1.5.5.7.1.1": "authorityInfoAccess",
	"1.3.6.1.4.1.11129.2.4.2": "signedCertificateTimestamps",
}

func describe(cert *x509.Certificate) certInfo {
	info := certInfo{Subject: cert.Subject.String(), Issuer: cert.Issuer.String(),
		Signature: cert.SignatureAlgorithm.String(), DNSNames: cert.DNSNames}
	switch k := cert.PublicKey.(type) {
	case *rsa.PublicKey:
		info.Key = fmt.Sprintf("RSA %d", k.N.BitLen())
	case *ecdsa.PublicKey:
		info.Key = "ECDSA " + k.Curve.Params().Name
	case ed25519.PublicKey:
		info.Key = "Ed25519"
	default:
		info.Key = cert.PublicKeyAlgorithm.String()
	}
	for _, e := range cert.Extensions {
		name := cmp.Or(extensionNames[e.Id.String()], e.Id.String())
		if e.Critical {
			name += " (critical)"
		}
		info.Extensions = append(info.Extensions, name)
	}
	return info
}

// h2Settings opens an HTTP/2 connection and records the server's SETTINGS and its connection-level
// WINDOW_UPDATE: until it is silent for a second, or for three seconds at most.
func h2Settings(sni, addr string) (*h2Result, error) {
	time.Sleep(probePause)
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	conn, err := tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{
		ServerName:         sni,
		NextProtos:         []string{"h2"},
		InsecureSkipVerify: true, // #nosec G402 -- the profile, not trust
	})
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	res := &h2Result{ALPN: conn.ConnectionState().NegotiatedProtocol, Settings: map[string]uint32{}}
	if res.ALPN != "h2" {
		return res, nil
	}
	if _, err := io.WriteString(conn, http2.ClientPreface); err != nil {
		return nil, err
	}
	framer := http2.NewFramer(conn, conn)
	if err := framer.WriteSettings(); err != nil {
		return nil, err
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		next := time.Now().Add(time.Second)
		if next.After(deadline) {
			next = deadline
		}
		if err := conn.SetReadDeadline(next); err != nil {
			return nil, err
		}
		frame, err := framer.ReadFrame()
		if err != nil {
			break // silence, the deadline, or the server closing
		}
		switch f := frame.(type) {
		case *http2.SettingsFrame:
			if f.IsAck() {
				continue
			}
			_ = f.ForeachSetting(func(s http2.Setting) error {
				res.Settings[s.ID.String()] = s.Val
				return nil
			})
			if err := framer.WriteSettingsAck(); err != nil {
				return nil, err
			}
		case *http2.WindowUpdateFrame:
			if f.StreamID == 0 && res.WindowUpdate == nil {
				res.WindowUpdate = &f.Increment
			}
		}
	}
	return res, nil
}
