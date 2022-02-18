package ptx

import (
	"context"
	"errors"
	"net"
	"sync"

	sflib "git.torproject.org/pluggable-transports/snowflake.git/v2/client/lib"
	"git.torproject.org/pluggable-transports/snowflake.git/v2/common/event"
	"github.com/ooni/probe-cli/v3/internal/stuninput"
)

// SnowflakeRendezvousMethod is the method which with we perform the rendezvous.
type SnowflakeRendezvousMethod interface {
	// Name is the name of the method.
	Name() string

	// AMPCacheURL returns a suitable AMP cache URL.
	AMPCacheURL() string

	// BrokerURL returns a suitable broker URL.
	BrokerURL() string

	// FrontDomain returns a suitable front domain.
	FrontDomain() string
}

// NewSnowflakeRendezvousMethodDomainFronting is a rendezvous method
// that uses domain fronting to perform the rendezvous.
func NewSnowflakeRendezvousMethodDomainFronting() SnowflakeRendezvousMethod {
	return &snowflakeRendezvousMethodDomainFronting{}
}

type snowflakeRendezvousMethodDomainFronting struct{}

func (d *snowflakeRendezvousMethodDomainFronting) Name() string {
	return "domain_fronting"
}

func (d *snowflakeRendezvousMethodDomainFronting) AMPCacheURL() string {
	return ""
}

func (d *snowflakeRendezvousMethodDomainFronting) BrokerURL() string {
	return "https://snowflake-broker.torproject.net.global.prod.fastly.net/"
}

func (d *snowflakeRendezvousMethodDomainFronting) FrontDomain() string {
	return "cdn.sstatic.net"
}

// NewSnowflakeRendezvousMethodAMP is a rendezvous method that
// uses the AMP cache to perform the rendezvous.
func NewSnowflakeRendezvousMethodAMP() SnowflakeRendezvousMethod {
	return &snowflakeRendezvousMethodAMP{}
}

type snowflakeRendezvousMethodAMP struct{}

func (d *snowflakeRendezvousMethodAMP) Name() string {
	return "amp"
}

func (d *snowflakeRendezvousMethodAMP) AMPCacheURL() string {
	return "https://cdn.ampproject.org/"
}

func (d *snowflakeRendezvousMethodAMP) BrokerURL() string {
	return "https://snowflake-broker.torproject.net/"
}

func (d *snowflakeRendezvousMethodAMP) FrontDomain() string {
	return "www.google.com"
}

// ErrSnowflakeNoSuchRendezvousMethod indicates the given rendezvous
// method is not supported by this implementation.
var ErrSnowflakeNoSuchRendezvousMethod = errors.New("ptx: unsupported rendezvous method")

// NewSnowflakeRendezvousMethod creates a new rendezvous method by name. We currently
// support the following rendezvous methods:
//
// 1. "domain_fronting" uses domain fronting with the sstatic.net CDN;
//
// 2. "" means default and it is currently equivalent to "domain_fronting" (but
// we don't guarantee that this default may change over time);
//
// 3. "amp" uses the AMP cache.
//
// Returns either a valid rendezvous method or an error.
func NewSnowflakeRendezvousMethod(method string) (SnowflakeRendezvousMethod, error) {
	switch method {
	case "domain_fronting", "":
		return NewSnowflakeRendezvousMethodDomainFronting(), nil
	case "amp":
		return NewSnowflakeRendezvousMethodAMP(), nil
	default:
		return nil, ErrSnowflakeNoSuchRendezvousMethod
	}
}

// SnowflakeDialer is a dialer for snowflake. You SHOULD either use a factory
// for constructing this type or set the fields marked as MANDATORY.
type SnowflakeDialer struct {
	// RendezvousMethod is the MANDATORY rendezvous method to use.
	RendezvousMethod SnowflakeRendezvousMethod

	// newClientTransport is an OPTIONAL hook for creating
	// an alternative snowflakeTransport in testing.
	newClientTransport func(config sflib.ClientConfig) (snowflakeTransport, error)
}

// NewSnowflakeDialer creates a SnowflakeDialer with default settings.
func NewSnowflakeDialer() *SnowflakeDialer {
	return &SnowflakeDialer{
		RendezvousMethod:   NewSnowflakeRendezvousMethodDomainFronting(),
		newClientTransport: nil,
	}
}

// NewSnowflakeDialerWithRendezvousMethod creates a SnowflakeDialer
// using the given RendezvousMethod explicitly.
func NewSnowflakeDialerWithRendezvousMethod(m SnowflakeRendezvousMethod) *SnowflakeDialer {
	return &SnowflakeDialer{
		RendezvousMethod:   m,
		newClientTransport: nil,
	}
}

// snowflakeTransport is anything that allows us to dial a snowflake
type snowflakeTransport interface {
	// Dial dials a snowflake connection.
	Dial() (net.Conn, error)

	// AddSnowflakeEventListener adds an event listener.
	AddSnowflakeEventListener(receiver event.SnowflakeEventReceiver)

	// RemoveSnowflakeEventListener removes an event listener.
	RemoveSnowflakeEventListener(receiver event.SnowflakeEventReceiver)
}

// DialContext establishes a connection with the given SF proxy. The context
// argument allows to interrupt this operation midway.
func (d *SnowflakeDialer) DialContext(ctx context.Context) (net.Conn, error) {
	conn, _, err := d.dialContext(ctx)
	return conn, err
}

func (d *SnowflakeDialer) dialContext(
	ctx context.Context) (net.Conn, chan interface{}, error) {
	done := make(chan interface{})
	txp, err := d.newSnowflakeClient(sflib.ClientConfig{
		BrokerURL:          d.RendezvousMethod.BrokerURL(),
		AmpCacheURL:        d.RendezvousMethod.AMPCacheURL(),
		FrontDomain:        d.RendezvousMethod.FrontDomain(),
		ICEAddresses:       d.iceAddresses(),
		KeepLocalAddresses: false,
		Max:                d.maxSnowflakes(),
	})
	if err != nil {
		return nil, nil, err
	}
	connch, errch := make(chan net.Conn), make(chan error, 1)
	go func() {
		defer close(done) // allow tests to synchronize with this goroutine's exit
		evr := d.newEventReceiver()
		log.Println("****** snowflake: adding event listener")
		defer txp.RemoveSnowflakeEventListener(evr)
		txp.AddSnowflakeEventListener(evr)
		conn, err := txp.Dial()
		if err != nil {
			errch <- err // buffered channel
			return
		}
		select {
		case connch <- conn:
		default:
			conn.Close() // context won the race
		}
	}()
	select {
	case conn := <-connch:
		return conn, done, nil
	case err := <-errch:
		return nil, done, err
	case <-ctx.Done():
		return nil, done, ctx.Err()
	}
}

// newSnowflakeClient allows us to call a mock rather than
// the real sflib.NewSnowflakeClient.
func (d *SnowflakeDialer) newSnowflakeClient(
	config sflib.ClientConfig) (snowflakeTransport, error) {
	if d.newClientTransport != nil {
		return d.newClientTransport(config)
	}
	return &snowflakeEventIgnorer{}
}

// iceAddresses returns suitable ICE addresses.
func (d *SnowflakeDialer) iceAddresses() []string {
	return stuninput.AsSnowflakeInput()
}

// maxSnowflakes returns the number of snowflakes to collect.
func (d *SnowflakeDialer) maxSnowflakes() int {
	return 1
}

// AsBridgeArgument returns the argument to be passed to
// the tor command line to declare this bridge.
func (d *SnowflakeDialer) AsBridgeArgument() string {
	return "snowflake 192.0.2.3:1 2B280B23E1107BB62ABFC40DDCC8824814F80A72"
}

// Name returns the pluggable transport name.
func (d *SnowflakeDialer) Name() string {
	return "snowflake"
}
