package tcprunner

import (
	"bufio"
	"context"
	"crypto/tls"
	"net"
	"strings"

	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/tracex"
)

// Model describes a type that does a DNS lookup(s), then attempts several TCP sessions
type Model interface {
	// Stores the provided hostname
	Hostname(string)
	// Store DNS query result
	DNSResults([]*model.ArchivalDNSLookupResult)
	// Indicates one or more steps failed (can be overwritten)
	Failed(string)
	// Stores a new individual test key (for a TCP session) and returns a pointer to it
	NewRun(string, string) TCPSessionModel
}

// TCPSessionModel describes a type that does a single TCP connection and TLS handshake with a given IP/Port combo
type TCPSessionModel interface {
	// Store IP/port address used for this session
	IPPort(string, string)
	// Store TCP connect result
	ConnectResults([]*model.ArchivalTCPConnectResult)
	// Store TLS handshake result
	HandshakeResult(*model.ArchivalTLSOrQUICHandshakeResult)
	// Indicates a failure string, as well as an identifier for the failed step
	FailedStep(string, string)
}

// TCPRunner manages sequential TCP sessions to the same hostname (over different IPs)
type TCPRunner struct {
	Tk        Model
	Trace     *measurexlite.Trace
	Logger    model.Logger
	Ctx       context.Context
	Tlsconfig *tls.Config
}

// TCPSession Manages a single TCP session and TLS handshake to a given ip:port
type TCPSession struct {
	Itk     TCPSessionModel
	Runner  *TCPRunner
	Addr    string
	Port    string
	TLS     bool
	RawConn *net.Conn
	TLSConn *net.Conn
}

// FailedStep saves a failure (with an associated failed step identifier) into IndividualTestKeys
func (s *TCPSession) FailedStep(failure string, step string) {
	// Save FailedStep inside ITK
	s.Itk.FailedStep(failure, step)
	// Copy FailedStep to global TK
	s.Runner.Tk.Failed(failure)
	// Print the warning message
	s.Runner.Logger.Warn(failure)
}

// Close closes the open TCP connections
func (s *TCPSession) Close() {
	if s.TLS {
		var conn = *s.TLSConn
		conn.Close()
	} else {
		// TODO: should raw connection be closed anyway?
		var conn = *s.RawConn
		conn.Close()
	}
}

// CurrentConn returns the currently active connection (TLS or plaintext)
func (s *TCPSession) CurrentConn() net.Conn {
	if s.TLS {
		// TODO: move to Debugf
		s.Runner.Logger.Infof("Reusing TLS connection")
		return *s.TLSConn
	}
	s.Runner.Logger.Infof("Reusing plaintext connection")
	return *s.RawConn
}

// Conn initializes a new Run and IndividualTestKeys
func (r *TCPRunner) Conn(addr string, port string) (*TCPSession, bool) {
	// Get new individual test keys
	itk := r.Tk.NewRun(addr, port)

	s := new(TCPSession)
	s.Runner = r
	s.Itk = itk
	s.Addr = addr
	s.Port = port
	s.TLS = false

	if !s.Conn(addr, port) {
		return nil, false
	}
	return s, true
}

// Conn starts a new TCP/IP connection to addr/port
func (s *TCPSession) Conn(addr string, port string) bool {
	dialer := s.Runner.Trace.NewDialerWithoutResolver(s.Runner.Logger)
	s.Runner.Logger.Infof("Dialing to %s:%s", addr, port)
	conn, err := dialer.DialContext(s.Runner.Ctx, "tcp", net.JoinHostPort(addr, port))
	s.Itk.ConnectResults(s.Runner.Trace.TCPConnects())
	if err != nil {
		s.FailedStep(*tracex.NewFailure(err), "tcp_connect")
		return false
	}
	s.RawConn = &conn

	return true
}

// Resolve resolves a hostname to a list of addresses
func (r *TCPRunner) Resolve(host string) ([]string, bool) {
	r.Logger.Infof("Resolving DNS for %s", host)
	resolver := r.Trace.NewStdlibResolver(r.Logger)
	addrs, err := resolver.LookupHost(r.Ctx, host)
	r.Tk.DNSResults(r.Trace.DNSLookupsFromRoundTrip())
	if err != nil {
		r.Tk.Failed(*tracex.NewFailure(err))
		return []string{}, false
	}
	r.Logger.Infof("Finished DNS for %s: %v", host, addrs)

	return addrs, true
}

// Handshake performs a TLS handshake over the currently active connection
func (s *TCPSession) Handshake() bool {
	if s.TLS {
		// TLS already initialized...
		return true
	}
	s.Runner.Logger.Infof("Starting TLS handshake with %s:%s", s.Addr, s.Port)
	thx := s.Runner.Trace.NewTLSHandshakerStdlib(s.Runner.Logger)
	tconn, _, err := thx.Handshake(s.Runner.Ctx, *s.RawConn, s.Runner.Tlsconfig)
	s.Itk.HandshakeResult(s.Runner.Trace.FirstTLSHandshakeOrNil())
	if err != nil {
		s.FailedStep(*tracex.NewFailure(err), "tls_handshake")
		return false
	}

	s.TLS = true
	s.TLSConn = &tconn
	s.Runner.Logger.Infof("Handshake succeeded")
	return true
}

// StartTLS performs a StartTLS exchange by sending a message over the plaintext connection, waiting for a specific
// response, then performing a TLS handshake
func (s *TCPSession) StartTLS(message string, waitForResponse string) bool {
	if s.TLS {
		s.Runner.Logger.Warn("Requested TCPSession to do StartTLS when TLS is already enabled")
		return true
	}

	if message != "" {
		s.Runner.Logger.Infof("Asking for StartTLS upgrade")
		s.CurrentConn().Write([]byte(message))
	}

	if waitForResponse != "" {
		s.Runner.Logger.Infof("Waiting for server response containing: %s", waitForResponse)
		conn := s.CurrentConn()
		for {
			line, err := bufio.NewReader(conn).ReadString('\n')
			if err != nil {
				s.FailedStep(*tracex.NewFailure(err), "starttls_wait_ok")
				return false
			}
			s.Runner.Logger.Debugf("Received: %s", line)
			if strings.Contains(line, waitForResponse) {
				s.Runner.Logger.Infof("Server is ready for StartTLS")
				break
			}
		}
	}

	return s.Handshake()
}
