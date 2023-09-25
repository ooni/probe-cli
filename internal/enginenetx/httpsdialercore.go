package enginenetx

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"sync/atomic"
	"time"

	"github.com/ooni/probe-cli/v3/internal/logx"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// HTTPSDialerTactic is a tactic to establish a TLS connection.
type HTTPSDialerTactic struct {
	// Endpoint is the TCP endpoint to use for dialing.
	Endpoint string

	// InitialDelay is the time in nanoseconds after which
	// you would like to start this policy.
	InitialDelay time.Duration

	// SNI is the TLS ServerName to send over the wire.
	SNI string

	// VerifyHostname is the hostname using during
	// the X.509 certificate verification.
	VerifyHostname string
}

var _ fmt.Stringer = &HTTPSDialerTactic{}

// Clone makes a deep copy of this [HTTPSDialerTactic].
func (dt *HTTPSDialerTactic) Clone() *HTTPSDialerTactic {
	return &HTTPSDialerTactic{
		Endpoint:       dt.Endpoint,
		InitialDelay:   dt.InitialDelay,
		SNI:            dt.SNI,
		VerifyHostname: dt.VerifyHostname,
	}
}

// String implements fmt.Stringer.
func (dt *HTTPSDialerTactic) String() string {
	return string(runtimex.Try1(json.Marshal(dt)))
}

// Summary returns a string summarizing this [HTTPSDialerTactic] for the
// specific purpose of inserting the struct into a map.
//
// The fields used to compute the summary are:
//
// - IPAddr
//
// - SNI
//
// - VerifyHostname
//
// The returned string contains the above fields separated by space.
func (dt *HTTPSDialerTactic) Summary() string {
	return fmt.Sprintf("%v sni=%v verify=%v", dt.Endpoint, dt.SNI, dt.VerifyHostname)
}

// HTTPSDialerPolicy describes the policy used by the [*HTTPSDialer].
type HTTPSDialerPolicy interface {
	// LookupTactics returns zero or more tactics for the given host and port.
	LookupTactics(ctx context.Context, domain, port string) <-chan *HTTPSDialerTactic
}

// HTTPSDialerStatsTracker tracks what happens while dialing TLS connections.
type HTTPSDialerStatsTracker interface {
	// These callbacks are invoked during the TLS handshake to inform this
	// tactic about events that occurred. A tactic SHOULD keep track of which
	// addresses, SNIs, etc. work and return them more frequently.
	//
	// Callbacks that take an error as argument also take a context as
	// argument and MUST check whether the context has been canceled or
	// its timeout has expired (i.e., using ctx.Err()) to determine
	// whether the operation failed or was merely canceled. In the latter
	// case, obviously, the policy MUST NOT consider the tactic failed.
	OnStarting(tactic *HTTPSDialerTactic)
	OnTCPConnectError(ctx context.Context, tactic *HTTPSDialerTactic, err error)
	OnTLSHandshakeError(ctx context.Context, tactic *HTTPSDialerTactic, err error)
	OnTLSVerifyError(tactic *HTTPSDialerTactic, err error)
	OnSuccess(tactic *HTTPSDialerTactic)
}

// HTTPSDialer is the [model.TLSDialer] used by the engine to dial HTTPS connections.
//
// The zero value of this struct is invalid; construct using [NewHTTPSDialer].
//
// This dialer MAY use an happy-eyeballs-like policy where we may try several IP addresses,
// including IPv4 and IPv6, and dialing tactics in parallel.
type HTTPSDialer struct {
	// idGenerator is the ID generator.
	idGenerator *atomic.Int64

	// logger is the logger to use.
	logger model.Logger

	// netx is the [*netxlite.Netx] to use.
	netx *netxlite.Netx

	// policy defines the dialing policy to use.
	policy HTTPSDialerPolicy

	// rootCAs contains the root certificate pool we should use.
	rootCAs *x509.CertPool

	// stats tracks what happens while dialing.
	stats HTTPSDialerStatsTracker
}

// NewHTTPSDialer constructs a new [*HTTPSDialer] instance.
//
// Arguments:
//
// - logger is the logger to use for logging;
//
// - netx is the [*netxlite.Netx] to use;
//
// - policy defines the dialer policy;
//
// - stats tracks what happens while we're dialing.
//
// The returned [*HTTPSDialer] would use the underlying network's
// DefaultCertPool to create and cache the cert pool to use.
func NewHTTPSDialer(
	logger model.Logger,
	netx *netxlite.Netx,
	policy HTTPSDialerPolicy,
	stats HTTPSDialerStatsTracker,
) *HTTPSDialer {
	return &HTTPSDialer{
		idGenerator: &atomic.Int64{},
		logger: &logx.PrefixLogger{
			Prefix: "HTTPSDialer: ",
			Logger: logger,
		},
		netx:    netx,
		policy:  policy,
		rootCAs: netx.MaybeCustomUnderlyingNetwork().Get().DefaultCertPool(),
		stats:   stats,
	}
}

var _ model.TLSDialer = &HTTPSDialer{}

// CloseIdleConnections implements model.TLSDialer.
func (hd *HTTPSDialer) CloseIdleConnections() {
	// nothing
}

// httpsDialerErrorOrConn contains either an error or a valid conn.
type httpsDialerErrorOrConn struct {
	// Conn is the established TLS conn or nil.
	Conn model.TLSConn

	// Err is the error or nil.
	Err error
}

// errDNSNoAnswer is the error returned when we have no tactic to try
var errDNSNoAnswer = netxlite.NewErrWrapper(
	netxlite.ClassifyResolverError,
	netxlite.DNSRoundTripOperation,
	netxlite.ErrOODNSNoAnswer,
)

// DialTLSContext implements model.TLSDialer.
func (hd *HTTPSDialer) DialTLSContext(ctx context.Context, network string, endpoint string) (net.Conn, error) {
	hostname, port, err := net.SplitHostPort(endpoint)
	if err != nil {
		return nil, err
	}

	// We need a cancellable context to interrupt the tactics emitter early when we
	// immediately get a valid response and we don't need to use other tactics.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// The emitter will emit tactics and then close the channel when done. We spawn 1+ workers
	// that handle tactics in paralellel and posts on the collector channel.
	emitter := hd.policy.LookupTactics(ctx, hostname, port)
	collector := make(chan *httpsDialerErrorOrConn)
	joiner := make(chan any)
	const parallelism = 16
	for idx := 0; idx < parallelism; idx++ {
		go hd.worker(ctx, joiner, emitter, collector)
	}

	// wait until all goroutines have joined
	var (
		connv     = []model.TLSConn{}
		errorv    = []error{}
		numJoined = 0
	)
	for numJoined < parallelism {
		select {
		case <-joiner:
			numJoined++

		case result := <-collector:
			// If the goroutine failed, record the error and continue processing results
			if result.Err != nil {
				errorv = append(errorv, result.Err)
				continue
			}

			// Save the conn and tell goroutines to stop ASAP
			connv = append(connv, result.Conn)
			cancel()
		}
	}

	return httpsDialerReduceResult(connv, errorv)
}

// httpsDialerReduceResult returns either an established conn or an error, using [errDNSNoAnswer] in
// case the list of connections and the list of errors are empty.
func httpsDialerReduceResult(connv []model.TLSConn, errorv []error) (model.TLSConn, error) {
	switch {
	case len(connv) >= 1:
		for _, c := range connv[1:] {
			c.Close()
		}
		return connv[0], nil

	case len(errorv) >= 1:
		return nil, errors.Join(errorv...)

	default:
		return nil, errDNSNoAnswer
	}
}

// worker attempts to establish a TLS connection using and emits a single
// [*httpsDialerErrorOrConn] for each tactic.
func (hd *HTTPSDialer) worker(ctx context.Context, joiner chan<- any,
	reader <-chan *HTTPSDialerTactic, writer chan<- *httpsDialerErrorOrConn) {
	// let the parent know that we terminated
	defer func() { joiner <- true }()

	for tactic := range reader {
		prefixLogger := &logx.PrefixLogger{
			Prefix: fmt.Sprintf("[#%d] ", hd.idGenerator.Add(1)),
			Logger: hd.logger,
		}

		conn, err := hd.dialTLS(ctx, prefixLogger, tactic)

		writer <- &httpsDialerErrorOrConn{Conn: conn, Err: err}
	}
}

// dialTLS performs the actual TLS dial.
func (hd *HTTPSDialer) dialTLS(
	ctx context.Context, logger model.Logger, tactic *HTTPSDialerTactic) (model.TLSConn, error) {
	// wait for the tactic to be ready to run
	if err := httpsDialerTacticWaitReady(ctx, tactic); err != nil {
		return nil, err
	}

	// tell the tactic that we're starting
	hd.stats.OnStarting(tactic)

	// create dialer and establish TCP connection
	ol := logx.NewOperationLogger(logger, "TCPConnect %s", tactic.Endpoint)
	dialer := hd.netx.NewDialerWithoutResolver(logger)
	tcpConn, err := dialer.DialContext(ctx, "tcp", tactic.Endpoint)
	ol.Stop(err)

	// handle a dialing error
	if err != nil {
		hd.stats.OnTCPConnectError(ctx, tactic, err)
		return nil, err
	}

	// create TLS configuration
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true, // Note: we're going to verify at the end of the func
		NextProtos:         []string{"h2", "http/1.1"},
		RootCAs:            hd.rootCAs,
		ServerName:         tactic.SNI,
	}

	// create handshaker and establish a TLS connection
	ol = logx.NewOperationLogger(
		logger,
		"TLSHandshake with %s SNI=%s ALPN=%v",
		tactic.Endpoint,
		tlsConfig.ServerName,
		tlsConfig.NextProtos,
	)
	thx := hd.netx.NewTLSHandshakerStdlib(logger)
	tlsConn, err := thx.Handshake(ctx, tcpConn, tlsConfig)
	ol.Stop(err)

	// handle handshake error
	if err != nil {
		hd.stats.OnTLSHandshakeError(ctx, tactic, err)
		tcpConn.Close()
		return nil, err
	}

	// verify the certificate chain
	ol = logx.NewOperationLogger(logger, "TLSVerifyCertificateChain %s", tactic.VerifyHostname)
	err = httpsDialerVerifyCertificateChain(tactic.VerifyHostname, tlsConn, hd.rootCAs)
	ol.Stop(err)

	// handle verification error
	if err != nil {
		hd.stats.OnTLSVerifyError(tactic, err)
		tlsConn.Close()
		return nil, err
	}

	// make sure the tactic know it worked
	hd.stats.OnSuccess(tactic)

	return tlsConn, nil
}

// httpsDialerWaitReady waits for the given delay to expire or the context to be canceled. If the
// delay is zero or negative, we immediately return nil. We also return nil when the delay expires. We
// return the context error if the context expires.
func httpsDialerTacticWaitReady(ctx context.Context, tactic *HTTPSDialerTactic) error {
	delay := tactic.InitialDelay
	if delay <= 0 {
		return nil
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-timer.C:
		return nil

	case <-ctx.Done():
		return netxlite.NewTopLevelGenericErrWrapper(ctx.Err())
	}
}

// errNoPeerCertificate is an internal error returned when we don't have any peer certificate.
var errNoPeerCertificate = errors.New("no peer certificate")

// errEmptyVerifyHostname indicates there is no hostname to verify against
var errEmptyVerifyHostname = errors.New("empty VerifyHostname")

// httpsDialerVerifyCertificateChain verifies the certificate chain with the given hostname.
func httpsDialerVerifyCertificateChain(hostname string, conn model.TLSConn, rootCAs *x509.CertPool) error {
	// This code comes from the example in the Go source tree that shows
	// how to override certificate verification and which is advertised
	// as follows:
	//
	//	VerifyConnection can be used to replace and customize connection
	//	verification. This example shows a VerifyConnection implementation that
	//	will be approximately equivalent to what crypto/tls does normally to
	//	verify the peer's certificate.
	//
	// See https://github.com/golang/go/blob/go1.21.0/src/crypto/tls/example_test.go#L186
	//
	// As of go1.21.0, the code we're replacing has approximately the same
	// implementation of the verification code we added below.
	//
	// See https://github.com/golang/go/blob/go1.21.0/src/crypto/tls/handshake_client.go#L962.

	// Protect against a programming or configuration error where the
	// programmer or user has not set the hostname.
	if hostname == "" {
		return errEmptyVerifyHostname
	}

	state := conn.ConnectionState()
	opts := x509.VerifyOptions{
		DNSName:       hostname, // note: here we're using the real hostname
		Intermediates: x509.NewCertPool(),
		Roots:         rootCAs,
	}

	// The following check is rather paranoid and it's not part of the Go codebase
	// from which we copied it, but I think it's important to be defensive.
	//
	// Because of that, I don't want to just drop an assertion here.
	if len(state.PeerCertificates) < 1 {
		return errNoPeerCertificate
	}

	for _, cert := range state.PeerCertificates[1:] {
		opts.Intermediates.AddCert(cert)
	}

	if _, err := state.PeerCertificates[0].Verify(opts); err != nil {
		return netxlite.NewErrWrapper(netxlite.ClassifyTLSHandshakeError, netxlite.TopLevelOperation, err)
	}
	return nil
}
