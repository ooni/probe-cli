package signalclient

import (
	"context"
	"crypto/x509"
	"errors"
	"net"
	"time"

	"github.com/ooni/probe-cli/v3/internal/logx"
	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

const (
	testName    = "signal_client" // provisional name, see doc.go
	testVersion = "0.1.0"
)

// Method is how a target is exercised after TLS.
type Method string

const (
	// MethodProvisioningH2 is libsignal's RFC 8441 extended CONNECT to the provisioning
	// WebSocket, followed by the provisioning flow.
	MethodProvisioningH2 Method = "provisioning_h2"

	// MethodProvisioningH1 is Android OkHttp's HTTP/1.1 WebSocket upgrade to the provisioning
	// WebSocket, followed by the provisioning flow.
	MethodProvisioningH1 Method = "provisioning_h1"

	// MethodEnclaveH1 is libsignal's HTTP/1.1 WebSocket upgrade to an enclave, without
	// credentials: 401 (current enclave id) or 404 (other ids) both mean Signal answered.
	MethodEnclaveH1 Method = "enclave_h1"

	// MethodHTTPSGet is one HTTPS GET of "/" (HTTP/2 when negotiated).
	MethodHTTPSGet Method = "https_get"
)

// Target is one connection a real Signal client makes.
type Target struct {
	Name     string
	Hostname string
	Hello    Hello
	Method   Method
	Path     string
}

// enclaveProbePath is a syntactically valid enclave path that names no real enclave, so the
// result never depends on the quarterly enclave ids.
const enclaveProbePath = "/v1/0000000000000000000000000000000000000000000000000000000000000000"

// DefaultTargets are Signal's Signal-hosted endpoints, each with the hello and protocol of the
// client that uses it (signal.md §1; hellos/README.md).
var DefaultTargets = []Target{
	{"chat_libsignal", "grpc.chat.signal.org", HelloLibsignal, MethodProvisioningH2, "/v1/websocket/provisioning/"},
	{"chat_android", "chat.signal.org", HelloAndroidWebSocket, MethodProvisioningH1, "/v1/websocket/provisioning/"},
	{"cdsi", "cdsi.signal.org", HelloLibsignalEnclave, MethodEnclaveH1, enclaveProbePath + "/discovery"},
	{"svr2", "svr2.signal.org", HelloLibsignalEnclave, MethodEnclaveH1, enclaveProbePath},
	{"svrb", "svrb.signal.org", HelloLibsignalEnclave, MethodEnclaveH1, enclaveProbePath},
	{"storage", "storage.signal.org", HelloAndroidREST, MethodHTTPSGet, "/"},
	{"cdn", "cdn.signal.org", HelloAndroidREST, MethodHTTPSGet, "/"},
	{"cdn2", "cdn2.signal.org", HelloAndroidREST, MethodHTTPSGet, "/"},
	{"cdn3", "cdn3.signal.org", HelloAndroidREST, MethodHTTPSGet, "/"},
	{"updates2", "updates2.signal.org", HelloAndroidREST, MethodHTTPSGet, "/"},
	{"updates2_desktop", "updates2.signal.org", HelloDesktop, MethodHTTPSGet, "/desktop/latest-linux.yml"},
	{"sfu", "sfu.voip.signal.org", HelloAndroidREST, MethodHTTPSGet, "/"},
}

// keepalivePaddings are the keepalive body sizes: none, message-sized, attachment-chunk-sized.
var keepalivePaddings = []int{0, 1024, 16384}

// Config contains the experiment config.
type Config struct {
	// IdleSeconds keeps each provisioning socket idle this long before the keepalives,
	// to detect middleboxes that kill idle sessions (libsignal pings after 31 s, drops after 45 s).
	IdleSeconds int64 `ooni:"seconds of idle time on provisioning sockets before keepalives (0 = none)"`

	// UserAgent is sent inside TLS, so it does not change what the network sees.
	UserAgent string `ooni:"User-Agent sent inside TLS"`
}

func (c *Config) userAgent() string {
	if c.UserAgent != "" {
		return c.UserAgent
	}
	return "ooniprobe-" + testName + "/" + testVersion
}

// TargetResult is the outcome for one target.
type TargetResult struct {
	Name            string              `json:"name"`
	Hostname        string              `json:"hostname"`
	Address         string              `json:"address"`
	ClientHello     string              `json:"client_hello"`
	Method          Method              `json:"method"`
	ALPN            string              `json:"alpn"`
	LeafKeyType     string              `json:"leaf_key_type"`
	Status          int64               `json:"status"`
	SignalTimestamp bool                `json:"x_signal_timestamp"`
	WSExtensions    string              `json:"ws_extensions,omitempty"`
	Provisioning    *ProvisioningResult `json:"provisioning,omitempty"`
	Failure         *string             `json:"failure"`
	FailedOperation string              `json:"failed_operation,omitempty"`
	T0              float64             `json:"t0"`
	T               float64             `json:"t"`
}

// TestKeys contains the experiment results.
type TestKeys struct {
	NetworkEvents []*model.ArchivalNetworkEvent             `json:"network_events"`
	Queries       []*model.ArchivalDNSLookupResult          `json:"queries"`
	TCPConnect    []*model.ArchivalTCPConnectResult         `json:"tcp_connect"`
	TLSHandshakes []*model.ArchivalTLSOrQUICHandshakeResult `json:"tls_handshakes"`
	Targets       []*TargetResult                           `json:"signal_targets"`

	// SignalServiceStatus is Signal's own outage beacon: uptime.signal.org resolves to
	// 127.0.0.1 when the service is up and 127.0.0.2 during an outage.
	SignalServiceStatus string `json:"signal_service_status"`
}

// Measurer performs the measurement.
type Measurer struct {
	config  Config
	targets []Target
	roots   func() (*x509.CertPool, error) // tests replace these
}

// NewExperimentMeasurer creates a new ExperimentMeasurer.
func NewExperimentMeasurer(config Config) model.ExperimentMeasurer {
	return &Measurer{config: config, targets: DefaultTargets, roots: signalRoots}
}

// ExperimentName implements model.ExperimentMeasurer.
func (m *Measurer) ExperimentName() string { return testName }

// ExperimentVersion implements model.ExperimentMeasurer.
func (m *Measurer) ExperimentVersion() string { return testVersion }

// Experiment-specific failure strings (in addition to df-007's).
const (
	// FailureExtendedConnectNotSupported: the HTTP/2 server did not offer RFC 8441 extended CONNECT.
	FailureExtendedConnectNotSupported = "extended_connect_not_supported"

	// FailureWebSocketUpgradeRefused: the server answered the WebSocket upgrade or extended
	// CONNECT with a status other than success (see the target's status).
	FailureWebSocketUpgradeRefused = "websocket_upgrade_refused"

	// FailureUnexpectedMessage: the server sent a WebSocket message the flow does not allow.
	FailureUnexpectedMessage = "unexpected_message"

	// FailureMalformedMessage: a WebSocket message was not a valid WebSocketMessage protobuf.
	FailureMalformedMessage = "malformed_message"
)

// newFailure wraps err so that it is recorded with the given failure string.
func newFailure(failure, operation string, err error) error {
	return &netxlite.ErrWrapper{Failure: failure, Operation: operation, WrappedErr: err}
}

// errNoAddress means DNS returned no usable address.
var errNoAddress = newFailure(netxlite.FailureDNSNoAnswer, netxlite.ResolveOperation, errors.New("no usable address"))

// Run implements model.ExperimentMeasurer.
func (m *Measurer) Run(ctx context.Context, args *model.ExperimentArgs) error {
	logger := args.Session.Logger()
	zero := args.Measurement.MeasurementStartTimeSaved
	tk := &TestKeys{}
	args.Measurement.TestKeys = tk
	roots, err := m.roots()
	if err != nil {
		return err
	}

	// Signal's outage beacon, a control: if Signal says it is down, failures mean little.
	trace := measurexlite.NewTrace(0, zero)
	addrs, err := trace.NewStdlibResolver(logger).LookupHost(ctx, "uptime.signal.org")
	tk.Queries = append(tk.Queries, trace.DNSLookupsFromRoundTrip()...)
	switch {
	case err != nil:
		tk.SignalServiceStatus = *measurexlite.NewFailure(err)
	case contains(addrs, "127.0.0.1"):
		tk.SignalServiceStatus = "ok"
	case contains(addrs, "127.0.0.2"):
		tk.SignalServiceStatus = "outage"
	default:
		tk.SignalServiceStatus = "unexpected_answer"
	}

	for i, target := range m.targets {
		tk.Targets = append(tk.Targets, m.measureTarget(ctx, logger, zero, int64(i+1), target, roots, tk))
	}
	return nil
}

func (m *Measurer) measureTarget(ctx context.Context, logger model.Logger, zero time.Time, index int64,
	target Target, roots *x509.CertPool, tk *TestKeys) *TargetResult {
	res := &TargetResult{Name: target.Name, Hostname: target.Hostname, ClientHello: target.Hello.Name,
		Method: target.Method, T0: time.Since(zero).Seconds()}
	defer func() { res.T = time.Since(zero).Seconds() }()
	fail := func(op string, err error) *TargetResult {
		res.FailedOperation, res.Failure = op, measurexlite.NewFailure(err)
		return res
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second+time.Duration(m.config.IdleSeconds)*time.Second)
	defer cancel()

	trace := measurexlite.NewTrace(index, zero, target.Name)
	ol := logx.NewOperationLogger(logger, "signal_client: %s %s", target.Name, target.Hostname)
	defer func() { ol.Stop(errorOrNil(res.Failure)) }()

	addrs, err := trace.NewStdlibResolver(logger).LookupHost(ctx, target.Hostname)
	tk.Queries = append(tk.Queries, trace.DNSLookupsFromRoundTrip()...)
	if err != nil {
		return fail(netxlite.ResolveOperation, err)
	}
	ip := firstIPv4(addrs)
	if ip == "" {
		return fail(netxlite.ResolveOperation, errNoAddress)
	}
	res.Address = net.JoinHostPort(ip, "443")

	tr, err := dialTLS(ctx, trace, logger, zero, res.Address, target.Hostname, target.Hello, roots)
	tk.TCPConnect = append(tk.TCPConnect, trace.TCPConnects()...)
	defer func() { tk.NetworkEvents = append(tk.NetworkEvents, trace.NetworkEvents()...) }()
	if tr != nil && tr.handshake != nil {
		tk.TLSHandshakes = append(tk.TLSHandshakes, tr.handshake)
	}
	if err != nil {
		op := netxlite.TLSHandshakeOperation
		if tr == nil {
			op = netxlite.ConnectOperation
		}
		return fail(op, err)
	}
	defer tr.conn.Close()
	res.ALPN, res.LeafKeyType = tr.alpn, tr.leafKey

	ua := m.config.userAgent()
	switch target.Method {
	case MethodHTTPSGet:
		status, ts, err := httpsGet(ctx, tr.conn, tr.alpn, target.Hostname, target.Path, ua)
		res.Status, res.SignalTimestamp = int64(status), ts
		if err != nil {
			return fail(netxlite.HTTPRoundTripOperation, err)
		}
		return res
	case MethodEnclaveH1, MethodProvisioningH1, MethodProvisioningH2:
		var up *upgradeResult
		if target.Method == MethodProvisioningH2 {
			up, err = websocketH2(ctx, tr.conn, target.Hostname, target.Path, ua+" libsignal/0.105.0")
		} else {
			up, err = websocketH1(ctx, tr.conn, target.Hostname, target.Path, ua)
		}
		if up != nil {
			res.Status, res.SignalTimestamp, res.WSExtensions = int64(up.status), up.signalTimestamp, up.extensions
		}
		if target.Method == MethodEnclaveH1 {
			// no credentials: 401 or 404 from the enclave front end is the expected answer
			if errors.Is(err, errUpgradeRefused) && (res.Status == 401 || res.Status == 404) {
				return res
			}
			if err == nil {
				up.closer.Close()
			}
			return fail("websocket_upgrade", err)
		}
		if err != nil {
			return fail("websocket_upgrade", err)
		}
		defer up.closer.Close()
		res.Provisioning = runProvisioning(ctx, up.ws, zero, keepalivePaddings, time.Duration(m.config.IdleSeconds)*time.Second)
		if res.Provisioning.Failure != nil {
			res.Failure, res.FailedOperation = res.Provisioning.Failure, res.Provisioning.FailedStep
		}
		return res
	}
	return fail(netxlite.TopLevelOperation, errors.New("signal_client: unknown method"))
}

func firstIPv4(addrs []string) string {
	for _, a := range addrs {
		if ip := net.ParseIP(a); ip != nil && ip.To4() != nil {
			return a
		}
	}
	if len(addrs) > 0 {
		return addrs[0]
	}
	return ""
}

func contains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}

func errorOrNil(failure *string) error {
	if failure == nil {
		return nil
	}
	return errors.New(*failure)
}
