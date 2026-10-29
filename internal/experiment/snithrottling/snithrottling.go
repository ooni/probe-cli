// Package snithrottling contains the sni_throttling experiment.
//
// The experiment downloads the same resource from the same server address
// several times, changing only the TLS SNI: "target" SNIs that a censor may
// throttle or block (by default YouTube's) and "control" SNIs that it should
// leave alone. Because the server, the request and the response bytes are the
// same, a large difference between the target and control download speeds
// points at SNI-based throttling (see docs/design/dd-007-throttling.md).
//
// By default the experiment uses Google front ends, which serve the
// www.youtube.com player script regardless of the SNI the client sends.
package snithrottling

import (
	"bufio"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/ooni/probe-cli/v3/internal/logx"
	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/throttling"
)

const (
	testName    = "sni_throttling"
	testVersion = "0.1.0"
)

// Config contains the experiment config.
type Config struct {
	// Host is the HTTP Host header and the domain resolved to find the server.
	Host string `json:"host,omitempty" ooni:"HTTP Host header and domain to resolve"`

	// Path is the resource to download. When empty and Host is the default
	// one, the experiment discovers the current YouTube player script.
	Path string `json:"path,omitempty" ooni:"path of the resource to download"`

	// FallbackHost is resolved when resolving Host fails.
	FallbackHost string `json:"fallback_host,omitempty" ooni:"domain to resolve when resolving host fails"`

	// Address is the OPTIONAL endpoint to use instead of resolving Host.
	Address string `json:"address,omitempty" ooni:"endpoint to use instead of resolving the host"`

	// TargetSNIs is the space separated list of SNIs that may be throttled.
	TargetSNIs string `json:"target_snis,omitempty" ooni:"space separated SNIs that may be throttled"`

	// ControlSNIs is the space separated list of SNIs used as controls.
	ControlSNIs string `json:"control_snis,omitempty" ooni:"space separated SNIs used as controls"`

	// MaxBytes is the maximum number of body bytes to download each time.
	MaxBytes int64 `json:"max_bytes,omitempty" ooni:"maximum body bytes to download each time"`

	// Timeout is the timeout in seconds of each download.
	Timeout int64 `json:"timeout,omitempty" ooni:"timeout in seconds of each download"`

	// Rounds is the number of times to download using each SNI.
	Rounds int64 `json:"rounds,omitempty" ooni:"number of downloads with each SNI"`
}

const defaultHost = "www.youtube.com"

func (c *Config) host() string {
	if c.Host != "" {
		return c.Host
	}
	return defaultHost
}

func (c *Config) fallbackHost() string {
	if c.FallbackHost != "" {
		return c.FallbackHost
	}
	return "www.google.com"
}

func (c *Config) targetSNIs() []string {
	if c.TargetSNIs != "" {
		return strings.Fields(c.TargetSNIs)
	}
	return []string{"www.youtube.com", "rr1---sn-4g5e6nzz.googlevideo.com"}
}

func (c *Config) controlSNIs() []string {
	if c.ControlSNIs != "" {
		return strings.Fields(c.ControlSNIs)
	}
	return []string{"www.google.com", "example.org"}
}

func (c *Config) maxBytes() int64 {
	if c.MaxBytes > 0 {
		return c.MaxBytes
	}
	return 1 << 20
}

func (c *Config) timeout() time.Duration {
	if c.Timeout > 0 {
		return time.Duration(c.Timeout) * time.Second
	}
	return 10 * time.Second
}

func (c *Config) rounds() int64 {
	if c.Rounds > 0 {
		return c.Rounds
	}
	return 1
}

// Download is the result of downloading the resource using a given SNI.
type Download struct {
	// Role is "target", "control" or "discovery".
	Role string `json:"role"`

	// SNI is the SNI we used.
	SNI string `json:"sni"`

	// Address is the endpoint we connected to.
	Address string `json:"address"`

	// Path is the requested path.
	Path string `json:"path"`

	// TCPConnect is the result of the TCP connect.
	TCPConnect *model.ArchivalTCPConnectResult `json:"tcp_connect"`

	// TLSHandshake is the result of the TLS handshake.
	TLSHandshake *model.ArchivalTLSOrQUICHandshakeResult `json:"tls_handshake"`

	// NetworkEvents contains the I/O events and the bytes received samples.
	NetworkEvents []*model.ArchivalNetworkEvent `json:"network_events"`

	// StatusCode is the HTTP status code (zero if we did not get a response).
	StatusCode int64 `json:"status_code"`

	// BodyLength is the number of body bytes received.
	BodyLength int64 `json:"body_length"`

	// T0 is when we sent the request, relative to the measurement start.
	T0 float64 `json:"t0"`

	// T is when we stopped reading the body, relative to the measurement start.
	T float64 `json:"t"`

	// GoodputKbps is BodyLength over T - T0 in kbit/s.
	GoodputKbps float64 `json:"goodput_kbps"`

	// Failure is the failure that stopped this download, if any.
	Failure *string `json:"failure"`

	// body contains the body when discovering the path.
	body []byte
}

// Verdict values for each target SNI.
const (
	VerdictOK           = "ok"
	VerdictThrottled    = "throttled"
	VerdictBlocked      = "blocked"
	VerdictInconclusive = "inconclusive"
)

// ThrottlingRatio is the target over control goodput ratio below which
// we say a target SNI is throttled (the 10x of dd-007).
const ThrottlingRatio = 0.1

// TestKeys contains the experiment results.
type TestKeys struct {
	// Queries contains the DNS lookups.
	Queries []*model.ArchivalDNSLookupResult `json:"queries"`

	// Host is the HTTP Host header we used.
	Host string `json:"host"`

	// Path is the path we downloaded.
	Path string `json:"path"`

	// Address is the endpoint we used.
	Address string `json:"address"`

	// Downloads contains all the downloads in the order we performed them.
	Downloads []*Download `json:"downloads"`

	// ControlGoodputKbps is the best control goodput (zero when no control succeeded).
	ControlGoodputKbps float64 `json:"control_goodput_kbps"`

	// Verdicts maps each target SNI to its verdict.
	Verdicts map[string]string `json:"verdicts"`

	// Ratios maps each target SNI to its median goodput over ControlGoodputKbps.
	Ratios map[string]float64 `json:"ratios"`

	// Failure is set when we could not run the measurement at all.
	Failure *string `json:"failure"`
}

// Measurer performs the measurement.
type Measurer struct {
	// Config contains the experiment settings.
	Config Config

	// shuffle randomizes the order of the downloads (tests replace it).
	shuffle func(n int, swap func(i, j int))
}

// ExperimentName implements model.ExperimentMeasurer.
func (m *Measurer) ExperimentName() string {
	return testName
}

// ExperimentVersion implements model.ExperimentMeasurer.
func (m *Measurer) ExperimentVersion() string {
	return testVersion
}

// errNoAddress means we could not find an address to connect to.
var errNoAddress = errors.New("sni_throttling: no address to connect to")

// errNoPath means we could not discover the resource to download.
var errNoPath = errors.New("sni_throttling: cannot discover the resource to download")

// Run implements model.ExperimentMeasurer.
func (m *Measurer) Run(ctx context.Context, args *model.ExperimentArgs) error {
	measurement := args.Measurement
	logger := args.Session.Logger()
	zeroTime := measurement.MeasurementStartTimeSaved
	config := &m.Config

	tk := &TestKeys{
		Queries:   []*model.ArchivalDNSLookupResult{},
		Host:      config.host(),
		Downloads: []*Download{},
		Verdicts:  map[string]string{},
		Ratios:    map[string]float64{},
	}
	measurement.TestKeys = tk

	var index int64
	nextIndex := func() int64 {
		index++
		return index
	}

	// 1. find the address to connect to
	tk.Address = config.Address
	if tk.Address == "" {
		for _, domain := range []string{config.host(), config.fallbackHost()} {
			addr, queries := lookupIPv4(ctx, nextIndex(), zeroTime, logger, domain)
			tk.Queries = append(tk.Queries, queries...)
			if addr != "" {
				tk.Address = net.JoinHostPort(addr, "443")
				break
			}
		}
	}
	if tk.Address == "" {
		tk.Failure = measurexlite.NewFailure(errNoAddress)
		return nil // submit the measurement
	}

	// 2. find the resource to download
	tk.Path = config.Path
	if tk.Path == "" && config.host() == defaultHost {
		for _, sni := range config.controlSNIs() {
			d := m.download(ctx, nextIndex(), zeroTime, logger, "discovery", sni, tk.Address, tk.Host, "/", 2<<20, true)
			tk.Downloads = append(tk.Downloads, d)
			if tk.Path = youtubePlayerPath(d.body); tk.Path != "" {
				break
			}
		}
		if tk.Path == "" {
			tk.Failure = measurexlite.NewFailure(errNoPath)
			return nil // submit the measurement
		}
	}
	if tk.Path == "" {
		tk.Path = "/"
	}

	// 3. download using each SNI in random order
	type job struct{ role, sni string }
	var jobs []job
	for _, sni := range config.targetSNIs() {
		jobs = append(jobs, job{"target", sni})
	}
	for _, sni := range config.controlSNIs() {
		jobs = append(jobs, job{"control", sni})
	}
	shuffle := m.shuffle
	if shuffle == nil {
		shuffle = rand.Shuffle
	}
	for round := int64(0); round < config.rounds(); round++ {
		shuffle(len(jobs), func(i, j int) { jobs[i], jobs[j] = jobs[j], jobs[i] })
		for _, j := range jobs {
			d := m.download(ctx, nextIndex(), zeroTime, logger, j.role, j.sni, tk.Address,
				tk.Host, tk.Path, config.maxBytes(), false)
			tk.Downloads = append(tk.Downloads, d)
		}
	}

	// 4. compare targets with controls
	tk.analyze(config.targetSNIs())
	return nil // submit the measurement
}

// lookupIPv4 resolves domain and returns its first IPv4 address, if any.
func lookupIPv4(ctx context.Context, index int64, zeroTime time.Time, logger model.Logger,
	domain string) (string, []*model.ArchivalDNSLookupResult) {
	trace := measurexlite.NewTrace(index, zeroTime)
	ol := logx.NewOperationLogger(logger, "[#%d] lookup %s", index, domain)
	ctx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()
	reso := trace.NewStdlibResolver(logger)
	addrs, err := reso.LookupHost(ctx, domain)
	ol.Stop(err)
	queries := trace.DNSLookupsFromRoundTrip()
	for _, addr := range addrs {
		if ip := net.ParseIP(addr); ip != nil && ip.To4() != nil {
			return addr, queries
		}
	}
	return "", queries
}

// youtubePlayerRe matches the YouTube player script path in the home page.
var youtubePlayerRe = regexp.MustCompile(`/s/player/[A-Za-z0-9_-]+/[A-Za-z0-9_./-]+/base\.js`)

// youtubePlayerPath returns the path of the YouTube player script in body.
func youtubePlayerPath(body []byte) string {
	return string(youtubePlayerRe.Find(body))
}

// verifyPublicChain checks that the server certificate chains to a trusted
// root, without checking the name: the server presents a certificate for the
// SNI, which may not be a name it serves (e.g. a control SNI).
func verifyPublicChain(certs []*x509.Certificate, roots *x509.CertPool) error {
	if len(certs) < 1 {
		return errors.New("sni_throttling: no server certificate")
	}
	intermediates := x509.NewCertPool()
	for _, cert := range certs[1:] {
		intermediates.AddCert(cert)
	}
	_, err := certs[0].Verify(x509.VerifyOptions{Roots: roots, Intermediates: intermediates})
	return err
}

// download downloads path from address using the given SNI and Host header.
func (m *Measurer) download(ctx context.Context, index int64, zeroTime time.Time, logger model.Logger,
	role, sni, address, host, path string, maxBytes int64, keepBody bool) *Download {
	d := &Download{
		Role:          role,
		SNI:           sni,
		Address:       address,
		Path:          path,
		NetworkEvents: []*model.ArchivalNetworkEvent{},
	}
	ctx, cancel := context.WithTimeout(ctx, m.Config.timeout())
	defer cancel()

	trace := measurexlite.NewTrace(index, zeroTime, role)
	sampler := throttling.NewSampler(trace)
	defer func() {
		d.NetworkEvents = append(d.NetworkEvents, trace.NetworkEvents()...)
		d.NetworkEvents = append(d.NetworkEvents, sampler.ExtractSamples()...)
		_ = sampler.Close()
	}()

	ol := logx.NewOperationLogger(logger, "[#%d] %s GET https://%s%s using %s and SNI %s",
		index, role, host, path, address, sni)

	conn, err := trace.NewDialerWithoutResolver(logger).DialContext(ctx, "tcp", address)
	d.TCPConnect = trace.FirstTCPConnectOrNil()
	if err != nil {
		d.Failure = measurexlite.NewFailure(err)
		ol.Stop(err)
		return d
	}
	defer conn.Close()

	tlsConfig := &tls.Config{
		NextProtos:         []string{"http/1.1"},
		ServerName:         sni,
		InsecureSkipVerify: true, // #nosec G402 - we verify the chain below
	}
	tlsConn, err := trace.NewTLSHandshakerStdlib(logger).Handshake(ctx, conn, tlsConfig)
	d.TLSHandshake = trace.FirstTLSHandshakeOrNil()
	if err != nil {
		d.Failure = measurexlite.NewFailure(err)
		ol.Stop(err)
		return d
	}
	defer tlsConn.Close()

	// we skipped verification because a control SNI may not match the server
	// certificate, but the chain must still be trusted: otherwise, we would be
	// comparing downloads from a server that is not the one we wanted
	certs := tlsConn.ConnectionState().PeerCertificates
	if err := verifyPublicChain(certs, netxlite.DefaultCertPool()); err != nil {
		err = netxlite.NewTopLevelGenericErrWrapper(err)
		d.Failure = measurexlite.NewFailure(err)
		ol.Stop(err)
		return d
	}

	// make sure reads and writes stop when the context is done
	if deadline, ok := ctx.Deadline(); ok {
		_ = tlsConn.SetDeadline(deadline)
	}

	req := fmt.Sprintf("GET %s HTTP/1.1\r\nHost: %s\r\nUser-Agent: %s\r\nAccept: */*\r\n"+
		"Accept-Language: en-US,en;q=0.9\r\nConnection: close\r\n\r\n", path, host, model.HTTPHeaderUserAgent)
	d.T0 = trace.TimeSince(zeroTime).Seconds()
	if _, err := tlsConn.Write([]byte(req)); err != nil {
		d.Failure = measurexlite.NewFailure(err)
		ol.Stop(err)
		return d
	}

	resp, err := http.ReadResponse(bufio.NewReader(tlsConn), nil)
	if err != nil {
		d.T = trace.TimeSince(zeroTime).Seconds()
		d.Failure = measurexlite.NewFailure(netxlite.NewTopLevelGenericErrWrapper(err))
		ol.Stop(err)
		return d
	}
	// Note: we do not close resp.Body because closing it reads the rest of the body,
	// which would download more than maxBytes. Closing the connection is enough.
	d.StatusCode = int64(resp.StatusCode)

	// StreamAllContext, unlike CopyContext, returns what it read before a timeout,
	// which is how a throttled download ends.
	body, err := netxlite.StreamAllContext(ctx, io.LimitReader(resp.Body, maxBytes))
	d.T = trace.TimeSince(zeroTime).Seconds()
	n := int64(len(body))
	d.BodyLength = n
	if keepBody {
		d.body = body
	}
	if elapsed := d.T - d.T0; elapsed > 0 {
		d.GoodputKbps = float64(n) * 8 / elapsed / 1000
	}
	if err != nil {
		d.Failure = measurexlite.NewFailure(netxlite.NewTopLevelGenericErrWrapper(err))
	}
	ol.Stop(err)
	return d
}

// minVerdictBytes is the number of body bytes a download needs before we
// trust its goodput: fewer bytes mostly measure latency, not throughput.
const minVerdictBytes = 64 << 10

// usable tells whether a download got a response we can compare.
func (d *Download) usable() bool {
	return d.TLSHandshake != nil && d.TLSHandshake.Failure == nil && d.StatusCode == 200
}

// analyze computes the verdict for each target SNI.
func (tk *TestKeys) analyze(targets []string) {
	for _, d := range tk.Downloads {
		if d.Role == "control" && d.usable() && d.Failure == nil && d.BodyLength >= minVerdictBytes {
			tk.ControlGoodputKbps = max(tk.ControlGoodputKbps, d.GoodputKbps)
		}
	}
	for _, sni := range targets {
		if tk.ControlGoodputKbps <= 0 {
			tk.Verdicts[sni] = VerdictInconclusive
			continue
		}
		var goodputs []float64
		blocked := 0
		for _, d := range tk.Downloads {
			if d.Role != "target" || d.SNI != sni {
				continue
			}
			if !d.usable() || d.BodyLength <= 0 {
				blocked++
				continue
			}
			goodputs = append(goodputs, d.GoodputKbps)
		}
		if len(goodputs) <= 0 {
			if blocked > 0 {
				tk.Verdicts[sni] = VerdictBlocked
			} else {
				tk.Verdicts[sni] = VerdictInconclusive
			}
			continue
		}
		ratio := median(goodputs) / tk.ControlGoodputKbps
		tk.Ratios[sni] = ratio
		if ratio < ThrottlingRatio {
			tk.Verdicts[sni] = VerdictThrottled
		} else {
			tk.Verdicts[sni] = VerdictOK
		}
	}
}

func median(values []float64) float64 {
	v := append([]float64{}, values...)
	sort.Float64s(v)
	if len(v)%2 == 1 {
		return v[len(v)/2]
	}
	return (v[len(v)/2-1] + v[len(v)/2]) / 2
}

// NewExperimentMeasurer creates a new ExperimentMeasurer.
func NewExperimentMeasurer(config Config) model.ExperimentMeasurer {
	return &Measurer{Config: config}
}

var _ model.MeasurementSummaryKeysProvider = &TestKeys{}

// SummaryKeys contains summary keys for this experiment.
type SummaryKeys struct {
	Throttled []string `json:"throttled"`
	Blocked   []string `json:"blocked"`
	IsAnomaly bool     `json:"-"`
}

// MeasurementSummaryKeys implements model.MeasurementSummaryKeysProvider.
func (tk *TestKeys) MeasurementSummaryKeys() model.MeasurementSummaryKeys {
	sk := &SummaryKeys{Throttled: []string{}, Blocked: []string{}}
	for sni, verdict := range tk.Verdicts {
		switch verdict {
		case VerdictThrottled:
			sk.Throttled = append(sk.Throttled, sni)
		case VerdictBlocked:
			sk.Blocked = append(sk.Blocked, sni)
		}
	}
	sort.Strings(sk.Throttled)
	sort.Strings(sk.Blocked)
	sk.IsAnomaly = len(sk.Throttled) > 0 || len(sk.Blocked) > 0
	return sk
}

// Anomaly implements model.MeasurementSummaryKeys.
func (sk *SummaryKeys) Anomaly() bool {
	return sk.IsAnomaly
}
