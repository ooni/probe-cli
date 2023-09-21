package enginenetx

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ooni/probe-cli/v3/internal/logx"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// HTTPSDialerTactic is a tactic to establish a TLS connection.
type HTTPSDialerTactic struct {
	// IPAddr is the IP address to use for dialing.
	IPAddr string

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

// String implements fmt.Stringer.
func (dt *HTTPSDialerTactic) String() string {
	return string(runtimex.Try1(json.Marshal(dt)))
}

// HTTPSDialerPolicy describes the policy used by the [*HTTPSDialer].
type HTTPSDialerPolicy interface {
	// LookupTactics performs a DNS lookup for the given domain using the given resolver and
	// returns either a list of tactics for dialing or an error.
	//
	// This function MUST NOT return an empty list and a nil error. If this happens the
	// code inside [HTTPSDialer] will PANIC.
	LookupTactics(ctx context.Context, domain string, reso model.Resolver) ([]*HTTPSDialerTactic, error)

	// Parallelism returns the number of goroutines to create when TLS dialing. The
	// [HTTPSDialer] will PANIC if the returned number is less than 1.
	Parallelism() int
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
	OnTLSVerifyError(ctz context.Context, tactic *HTTPSDialerTactic, err error)
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

	// policy defines the dialing policy to use.
	policy HTTPSDialerPolicy

	// resolver is the DNS resolver to use.
	resolver model.Resolver

	// rootCAs contains the root certificate pool we should use.
	rootCAs *x509.CertPool

	// stats tracks what happens while dialing.
	stats HTTPSDialerStatsTracker

	// unet is the underlying network.
	unet model.UnderlyingNetwork

	// wg is the wait group for knowing when all goroutines
	// started in the background joined (for testing).
	wg *sync.WaitGroup
}

// NewHTTPSDialer constructs a new [*HTTPSDialer] instance.
//
// Arguments:
//
// - logger is the logger to use for logging;
//
// - policy defines the dialer policy;
//
// - resolver is the resolver to use;
//
// - stats tracks what happens while we're dialing;
//
// - unet is the underlying network to use.
//
// The returned [*HTTPSDialer] would use the underlying network's
// DefaultCertPool to create and cache the cert pool to use.
func NewHTTPSDialer(
	logger model.Logger,
	policy HTTPSDialerPolicy,
	resolver model.Resolver,
	stats HTTPSDialerStatsTracker,
	unet model.UnderlyingNetwork,
) *HTTPSDialer {
	return &HTTPSDialer{
		idGenerator: &atomic.Int64{},
		logger: &logx.PrefixLogger{
			Prefix: "HTTPSDialer: ",
			Logger: logger,
		},
		policy:   policy,
		resolver: resolver,
		rootCAs:  unet.DefaultCertPool(),
		stats:    stats,
		unet:     unet,
		wg:       &sync.WaitGroup{},
	}
}

var _ model.TLSDialer = &HTTPSDialer{}

// WaitGroup returns the [*sync.WaitGroup] tracking the number of background goroutines,
// which is definitely useful in testing to make sure we join all the goroutines.
func (hd *HTTPSDialer) WaitGroup() *sync.WaitGroup {
	return hd.wg
}

// CloseIdleConnections implements model.TLSDialer.
func (hd *HTTPSDialer) CloseIdleConnections() {
	hd.resolver.CloseIdleConnections()
}

// httpsDialerErrorOrConn contains either an error or a valid conn.
type httpsDialerErrorOrConn struct {
	// Conn is the established TLS conn or nil.
	Conn model.TLSConn

	// Err is the error or nil.
	Err error
}

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

	logger := &logx.PrefixLogger{
		Prefix: fmt.Sprintf("[#%d] ", hd.idGenerator.Add(1)),
		Logger: hd.logger,
	}
	ol := logx.NewOperationLogger(logger, "LookupTactics: %s", hostname)
	tactics, err := hd.policy.LookupTactics(ctx, hostname, hd.resolver)
	if err != nil {
		ol.Stop(err)
		return nil, err
	}
	ol.Stop(tactics)
	runtimex.Assert(len(tactics) >= 1, "expected at least one tactic here")

	emitter := hd.tacticsEmitter(ctx, tactics...)
	collector := make(chan *httpsDialerErrorOrConn)

	parallelism := hd.policy.Parallelism()
	runtimex.Assert(parallelism >= 1, "expected parallelism to be >= 1")
	for idx := 0; idx < parallelism; idx++ {
		hd.wg.Add(1)
		go func() {
			defer hd.wg.Done()
			hd.worker(ctx, hostname, emitter, port, collector)
		}()
	}

	var (
		numDials = len(tactics)
		errorv   = []error{}
	)
	for idx := 0; idx < numDials; idx++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()

		case result := <-collector:
			if result.Err != nil {
				errorv = append(errorv, result.Err)
				continue
			}

			// Returning early cancels the context and this cancellation
			// causes other background goroutines to interrupt their long
			// running network operations or unblocks them while sending
			return result.Conn, nil
		}
	}

	return nil, errors.Join(errorv...)
}

// tacticsEmitter returns a channel closed once we have emitted all the tactics or the context is done.
func (hd *HTTPSDialer) tacticsEmitter(ctx context.Context, tactics ...*HTTPSDialerTactic) <-chan *HTTPSDialerTactic {
	out := make(chan *HTTPSDialerTactic)

	hd.wg.Add(1)
	go func() {
		defer hd.wg.Done()
		defer close(out)

		for _, tactic := range tactics {
			select {
			case out <- tactic:
				continue

			case <-ctx.Done():
				return
			}
		}
	}()

	return out
}

// worker attempts to establish a TLS connection using and emits a single
// [*httpsDialerErrorOrConn] for each tactic.
func (hd *HTTPSDialer) worker(
	ctx context.Context,
	hostname string,
	reader <-chan *HTTPSDialerTactic,
	port string,
	writer chan<- *httpsDialerErrorOrConn,
) {
	// Note: no need to be concerned with the wait group here because
	// we're managing it inside DialTLSContext so Add and Done live together

	for {
		select {
		case tactic, good := <-reader:
			if !good {
				// This happens when the emitter goroutine has closed the channel
				return
			}

			logger := &logx.PrefixLogger{
				Prefix: fmt.Sprintf("[#%d] ", hd.idGenerator.Add(1)),
				Logger: hd.logger,
			}
			conn, err := hd.dialTLS(ctx, logger, tactic, port)

			select {
			case <-ctx.Done():
				if conn != nil {
					conn.Close() // we own the connection
				}
				return

			case writer <- &httpsDialerErrorOrConn{Conn: conn, Err: err}:
				continue
			}

		case <-ctx.Done():
			return
		}
	}
}

// dialTLS performs the actual TLS dial.
func (hd *HTTPSDialer) dialTLS(ctx context.Context,
	logger model.Logger, tactic *HTTPSDialerTactic, port string) (model.TLSConn, error) {
	// wait for the tactic to be ready to run
	if err := httpsDialerTacticWaitReady(ctx, tactic); err != nil {
		return nil, err
	}

	// tell the tactic that we're starting
	hd.stats.OnStarting(tactic)

	// create a network abstraction using the underlying network
	netx := &netxlite.Netx{Underlying: hd.unet}

	// create dialer and establish TCP connection
	endpoint := net.JoinHostPort(tactic.IPAddr, port)
	ol := logx.NewOperationLogger(logger, "TCPConnect %s", endpoint)
	dialer := netx.NewDialerWithoutResolver(logger)
	tcpConn, err := dialer.DialContext(ctx, "tcp", endpoint)
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
		endpoint,
		tlsConfig.ServerName,
		tlsConfig.NextProtos,
	)
	thx := netx.NewTLSHandshakerStdlib(logger)
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
		//tactic.OnTLSVerifyError(ctx, err)
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
		return ctx.Err()
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
