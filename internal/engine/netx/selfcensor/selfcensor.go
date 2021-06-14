// Package selfcensor contains code that triggers censorship. We use
// this functionality to implement integration tests.
//
// The self censoring functionality is disabled by default. To enable it,
// call Enable with a JSON-serialized Spec structure as its argument.
//
// The following example causes NXDOMAIN to be returned for `dns.google`:
//
//     selfcensor.Enable(`{"PoisonSystemDNS":{"dns.google":["NXDOMAIN"]}}`)
//
// The following example blocks connecting to `8.8.8.8:443`:
//
//     selfcensor.Enable(`{"BlockedEndpoints":{"8.8.8.8:443":"REJECT"}}`)
//
// The following example blocks packets containing dns.google:
//
//     selfcensor.Enable(`{"BlockedFingerprints":{"dns.google":"RST"}}`)
//
// The documentation of the Spec structure contains further information on
// how to populate the JSON. Miniooni uses the `--self-censor-spec flag` to
// which you are supposed to pass a serialized JSON.
package selfcensor

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log"
	"net"
	"sync"
	"time"

	"github.com/ooni/probe-cli/v3/internal/atomicx"
)

// Spec indicates what self censorship techniques to implement.
type Spec struct {
	// PoisonSystemDNS allows you to change the behaviour of the system
	// DNS regarding specific domains. They keys are the domains and the
	// values are the IP addresses to return. If you set the values for
	// a domain to `[]string{"NXDOMAIN"}`, the system resolver will return
	// an NXDOMAIN response. If you set the values for a domain to
	// `[]string{"TIMEOUT"}` the system resolver will return "i/o timeout".
	PoisonSystemDNS map[string][]string

	// BlockedEndpoints allows you to block specific IP endpoints. The key is
	// `IP:port` to block. The format is the same of net.JoinHostPort. If
	// the value is "REJECT", then the connection attempt will fail with
	// ECONNREFUSED. If the value is "TIMEOUT", then the connector will return
	// claiming "i/o timeout". If the value is anything else, we will
	// perform a "REJECT".
	BlockedEndpoints map[string]string

	// BlockedFingerprints allows you to block packets whose body contains
	// specific fingerprints. Of course, the key is the fingerprint. If
	// the value is "RST", then the connection will be reset. If the value
	// is "TIMEOUT", then the code will return claiming "i/o timeout". If
	// the value is anything else, we will perform a "RST".
	BlockedFingerprints map[string]string
}

var (
	attempts *atomicx.Int64 = &atomicx.Int64{}
	enabled  *atomicx.Int64 = &atomicx.Int64{}
	mu       sync.Mutex
	spec     *Spec
)

// Enabled returns whether self censorship is enabled
func Enabled() bool {
	return enabled.Load() != 0
}

// Attempts returns the number of self censorship attempts so far. A self
// censorship attempt is defined as the code entering into the branch that
// _may_ perform self censorship. We expected to see this counter being
// equal to zero when Enabled() returns false.
func Attempts() int64 {
	return attempts.Load()
}

// Enable turns on the self censorship engine. This function returns
// an error if we cannot parse a Spec from the serialized JSON inside
// data. Each time you call Enable you overwrite the previous spec.
func Enable(data string) error {
	mu.Lock()
	defer mu.Unlock()
	s := new(Spec)
	if err := json.Unmarshal([]byte(data), s); err != nil {
		return err
	}
	spec = s
	enabled.Add(1)
	log.Printf("selfcensor: spec %+v", *spec)
	return nil
}

// MaybeEnable is like enable except that it does nothing in case
// the string provided as argument is an empty string.
func MaybeEnable(data string) (err error) {
	if data != "" {
		err = Enable(data)
	}
	return
}

// SystemResolver is a self-censoring system resolver. This resolver does
// not censor anything unless you call selfcensor.Enable().
type SystemResolver struct{}

// errTimeout indicates that a timeout error has occurred.
var errTimeout = errors.New("i/o timeout")

// LookupHost implements Resolver.LookupHost
func (r SystemResolver) LookupHost(ctx context.Context, hostname string) ([]string, error) {
	if enabled.Load() != 0 { // jumps not taken by default
		mu.Lock()
		defer mu.Unlock()
		attempts.Add(1)
		if spec.PoisonSystemDNS != nil {
			values := spec.PoisonSystemDNS[hostname]
			if len(values) == 1 && values[0] == "NXDOMAIN" {
				return nil, errors.New("no such host")
			}
			if len(values) == 1 && values[0] == "TIMEOUT" {
				return nil, errTimeout
			}
			if len(values) > 0 {
				return values, nil
			}
		}
		// FALLTHROUGH
	}
	return net.DefaultResolver.LookupHost(ctx, hostname)
}

// Network implements Resolver.Network
func (r SystemResolver) Network() string {
	return "system"
}

// Address implements Resolver.Address
func (r SystemResolver) Address() string {
	return ""
}

// SystemDialer is a self-censoring system dialer. This dialer does
// not censor anything unless you call selfcensor.Enable().
type SystemDialer struct{}

// defaultNetDialer is the dialer we use by default.
var defaultNetDialer = &net.Dialer{
	Timeout:   15 * time.Second,
	KeepAlive: 15 * time.Second,
}

// DefaultDialer is the dialer you should use in code that wants
// to take advantage of selfcensor capabilities.
var DefaultDialer = SystemDialer{}

// DialContext implements Dialer.DialContext
func (d SystemDialer) DialContext(
	ctx context.Context, network, address string) (net.Conn, error) {
	if enabled.Load() != 0 { // jumps not taken by default
		mu.Lock()
		defer mu.Unlock()
		attempts.Add(1)
		if spec.BlockedEndpoints != nil {
			action, ok := spec.BlockedEndpoints[address]
			if ok && action == "TIMEOUT" {
				return nil, errTimeout
			}
			if ok {
				switch network {
				case "tcp", "tcp4", "tcp6":
					return nil, errors.New("connection refused")
				default:
					// not applicable
				}
			}
		}
		if spec.BlockedFingerprints != nil {
			conn, err := defaultNetDialer.DialContext(ctx, network, address)
			if err != nil {
				return nil, err
			}
			return connWrapper{Conn: conn, closed: make(chan interface{}, 128),
				fingerprints: spec.BlockedFingerprints}, nil
		}
		// FALLTHROUGH
	}
	return defaultNetDialer.DialContext(ctx, network, address)
}

type connWrapper struct {
	net.Conn
	closed       chan interface{}
	fingerprints map[string]string
}

func (c connWrapper) Write(p []byte) (int, error) {
	// TODO(bassosimone): implement reassembly to workaround the
	// splitting of the ClientHello message.
	if _, err := c.match(p, len(p)); err != nil {
		return 0, err
	}
	return c.Conn.Write(p)
}

func (c connWrapper) match(p []byte, n int) (int, error) {
	p = p[:n] // trim
	for key, value := range c.fingerprints {
		if bytes.Index(p, []byte(key)) != -1 {
			if value == "TIMEOUT" {
				return 0, errTimeout
			}
			return 0, errors.New("connection_reset")
		}
	}
	return n, nil
}

func (c connWrapper) Close() error {
	// Implementation note: we will block here if we attempt to close
	// too many times and noone's reading. Because we have a large buffer,
	// and because this is integration testing code, that's fine.
	c.closed <- true
	return c.Conn.Close()
}
