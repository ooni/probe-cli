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

	http "github.com/ooni/oohttp"
	"github.com/ooni/probe-cli/v3/internal/logmodel"
	"github.com/ooni/probe-cli/v3/internal/logx"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// httpsDialerTactic is a tactic to establish a TLS connection.
type httpsDialerTactic struct {
	// Address is the IPv4/IPv6 address for dialing.
	Address string

	// InitialDelay is the time in nanoseconds after which
	// you would like to start this policy.
	InitialDelay time.Duration

	// Port is the TCP port for dialing.
	Port string

	// SNI is the TLS ServerName to send over the wire.
	SNI string

	// VerifyHostname is the hostname using during
	// the X.509 certificate verification.
	VerifyHostname string
}

var _ fmt.Stringer = &httpsDialerTactic{}

// Clone makes a deep copy of this [httpsDialerTactic].
func (dt *httpsDialerTactic) Clone() *httpsDialerTactic {
	return &httpsDialerTactic{
		Address:        dt.Address,
		InitialDelay:   dt.InitialDelay,
		Port:           dt.Port,
		SNI:            dt.SNI,
		VerifyHostname: dt.VerifyHostname,
	}
}

// String implements fmt.Stringer.
func (dt *httpsDialerTactic) String() string {
	return string(runtimex.Try1(json.Marshal(dt)))
}

// tacticSummaryKey returns a string summarizing the tactic's features.
//
// The fields used to compute the summary are:
//
// - IPAddr
//
// - Port
//
// - SNI
//
// - VerifyHostname
//
// The returned string contains the above fields separated by space with
// `sni=` before the SNI and `verify=` before the verify hostname.
//
// We should be careful not to change this format unless we also change the
// format version used by user policies and by the state management.
func (dt *httpsDialerTactic) tacticSummaryKey() string {
	return fmt.Sprintf(
		"%v sni=%v verify=%v",
		net.JoinHostPort(dt.Address, dt.Port),
		dt.SNI,
		dt.VerifyHostname,
	)
}

// domainEndpointKey returns a string consisting of the domain endpoint only.
//
// We always use the VerifyHostname and the Port to construct the domain endpoint.
func (dt *httpsDialerTactic) domainEndpointKey() string {
	return net.JoinHostPort(dt.VerifyHostname, dt.Port)
}

// httpsDialerPolicy is a policy used by the [*httpsDialer].
type httpsDialerPolicy interface {
	// LookupTactics emits zero or more tactics for the given host and port
	// through the returned channel, which is closed when done.
	LookupTactics(ctx context.Context, domain, port string) <-chan *httpsDialerTactic
}

// httpsDialerEventsHandler handles events occurring while we try dialing TLS.
type httpsDialerEventsHandler interface {
	// These callbacks are invoked during the TLS dialing to inform this
	// interface about events that occurred. A policy SHOULD keep track of which
	// addresses, SNIs, etc. work and return them more frequently.
	//
	// Callbacks that take an error as argument also take a context as
	// argument and MUST check whether the context has been canceled or
	// its timeout has expired (i.e., using ctx.Err()) to determine
	// whether the operation failed or was merely canceled. In the latter
	// case, obviously, you MUST NOT consider the tactic failed.
	OnStarting(tactic *httpsDialerTactic)
	OnTCPConnectError(ctx context.Context, tactic *httpsDialerTactic, err error)
	OnTLSHandshakeError(ctx context.Context, tactic *httpsDialerTactic, err error)
	OnTLSVerifyError(tactic *httpsDialerTactic, err error)
	OnSuccess(tactic *httpsDialerTactic)
}

// httpsDialer is the [model.TLSDialer] used by the engine to dial HTTPS connections.
//
// The zero value of this struct is invalid; construct using [newHTTPSDialer].
//
// This dialer MAY use an happy-eyeballs-like policy where we may try several IP addresses,
// including IPv4 and IPv6, and dialing tactics in parallel.
type httpsDialer struct {
	// dialTLSFn is the actual function used to perform the dial. The constructor
	// initializes it to dialTLS but you can override it with testing.
	dialTLSFn func(
		ctx context.Context,
		logger logmodel.Logger,
		t0 time.Time,
		tactic *httpsDialerTactic,
	) (http.TLSConn, error)

	// idGenerator is the ID generator.
	idGenerator *atomic.Int64

	// logger is the logger to use.
	logger model.Logger

	// netx is the [*netxlite.Netx] to use.
	netx *netxlite.Netx

	// policy defines the dialing policy to use.
	policy httpsDialerPolicy

	// rootCAs contains the root certificate pool we should use.
	rootCAs *x509.CertPool

	// stats tracks what happens while dialing.
	stats httpsDialerEventsHandler
}

// newHTTPSDialer constructs a new [*httpsDialer] instance.
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
// The returned [*httpsDialer] would use the underlying network's
// DefaultCertPool to create and cache the cert pool to use.
func newHTTPSDialer(
	logger model.Logger,
	netx *netxlite.Netx,
	policy httpsDialerPolicy,
	stats httpsDialerEventsHandler,
) *httpsDialer {
	dx := &httpsDialer{
		dialTLSFn:   nil, // set just below
		idGenerator: &atomic.Int64{},
		logger: &logx.PrefixLogger{
			Prefix: "httpsDialer: ",
			Logger: logger,
		},
		netx:    netx,
		policy:  policy,
		rootCAs: netx.MaybeCustomUnderlyingNetwork().Get().DefaultCertPool(),
		stats:   stats,
	}
	dx.dialTLSFn = dx.dialTLS
	return dx
}

var _ model.TLSDialer = &httpsDialer{}

// CloseIdleConnections implements model.TLSDialer.
func (hd *httpsDialer) CloseIdleConnections() {
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
func (hd *httpsDialer) DialTLSContext(ctx context.Context, network string, endpoint string) (net.Conn, error) {
	hostname, port, err := net.SplitHostPort(endpoint)
	if err != nil {
		return nil, err
	}

	// We need a cancellable context to interrupt the tactics emitter early when we
	// immediately get a valid response and we don't need to use other tactics.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// The emitter will emit tactics and then close the channel when done. We spawn 16 workers
	// that handle tactics in parallel and post results on the collector channel.
	emitter := httpsDialerFilterTactics(hd.policy.LookupTactics(ctx, hostname, port))
	collector := make(chan *httpsDialerErrorOrConn)
	joiner := make(chan any)
	const parallelism = 16
	t0 := time.Now()
	for idx := 0; idx < parallelism; idx++ {
		go hd.worker(ctx, joiner, emitter, t0, collector)
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

			// Save the conn
			connv = append(connv, result.Conn)

			// Interrupt other concurrent dialing attempts
			cancel()
		}
	}

	return httpsDialerReduceResult(connv, errorv)
}

// httpsDialerFilterTactics filters the tactics to:
//
// 1. be paranoid and filter out nil tactics if any;
//
// 2. avoid emitting duplicate tactics as part of the same run;
//
// 3. rewrite the happy eyeball delays.
//
// This function returns a channel where we emit the edited
// tactics, and which we clone when we're done.
func httpsDialerFilterTactics(input <-chan *httpsDialerTactic) <-chan *httpsDialerTactic {
	return filterAssignInitialDelays(filterOnlyKeepUniqueTactics(filterOutNilTactics(input)))
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

// worker attempts to establish a TLS connection and emits the result using
// a [*httpsDialerErrorOrConn] for each tactic, until there are no more tactics
// and the reader channel is closed. At which point it posts on joiner to let
// the parent know that this goroutine has done its job.
func (hd *httpsDialer) worker(
	ctx context.Context,
	joiner chan<- any,
	reader <-chan *httpsDialerTactic,
	t0 time.Time,
	writer chan<- *httpsDialerErrorOrConn,
) {
	// let the parent know that we terminated
	defer func() { joiner <- true }()

	for tactic := range reader {
		prefixLogger := &logx.PrefixLogger{
			Prefix: fmt.Sprintf("[#%d] ", hd.idGenerator.Add(1)),
			Logger: hd.logger,
		}

		// perform the dial through an indirect function call mockabled for testing
		conn, err := hd.dialTLSFn(ctx, prefixLogger, t0, tactic)

		// send results to the parent
		writer <- &httpsDialerErrorOrConn{Conn: conn, Err: err}
	}
}

// dialTLS performs the actual TLS dial.
func (hd *httpsDialer) dialTLS(
	ctx context.Context,
	logger model.Logger,
	t0 time.Time,
	tactic *httpsDialerTactic,
) (model.TLSConn, error) {
	// honor happy-eyeballs delays and wait for the tactic to be ready to run
	if err := httpsDialerTacticWaitReady(ctx, t0, tactic); err != nil {
		return nil, err
	}

	// tell the observer that we're starting
	hd.stats.OnStarting(tactic)

	// create dialer and establish TCP connection
	endpoint := net.JoinHostPort(tactic.Address, tactic.Port)
	ol := logx.NewOperationLogger(logger, "TCPConnect %s", endpoint)
	dialer := hd.netx.NewDialerWithoutResolver(logger)
	tcpConn, err := dialer.DialContext(ctx, "tcp", endpoint)
	ol.Stop(err)

	// handle a dialing error
	if err != nil {
		hd.stats.OnTCPConnectError(ctx, tactic, err)
		return nil, err
	}

	// create TLS configuration
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true, // Note: we're going to verify at the end of the func!
		NextProtos:         []string{"h2", "http/1.1"},
		RootCAs:            hd.rootCAs,
		ServerName:         tactic.SNI,
	}

	// create handshaker and establish a TLS connection
	ol = logx.NewOperationLogger(
		logger,
		"TLSHandshake with %s SNI=%s ALPN=%v",
		endpoint,
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

	// make sure the observer knows it worked
	hd.stats.OnSuccess(tactic)

	return tlsConn, nil
}

// httpsDialerWaitReady waits for the given delay to expire or the context to be canceled. If the
// delay is zero or negative, we immediately return nil. We also return nil when the delay expires. We
// return the context error if the context expires.
func httpsDialerTacticWaitReady(
	ctx context.Context,
	t0 time.Time,
	tactic *httpsDialerTactic,
) error {
	deadline := t0.Add(tactic.InitialDelay)
	delta := time.Until(deadline)
	if delta <= 0 {
		return nil
	}

	timer := time.NewTimer(delta)
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
