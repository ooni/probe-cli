package ptx

import (
	"context"
	"net"

	sflib "git.torproject.org/pluggable-transports/snowflake.git/client/lib"
	"github.com/ooni/probe-cli/v3/internal/stuninput"
)

// SnowflakeDialer is a dialer for snowflake. When optional fields are
// not specified, we use defaults from the snowflake repository.
type SnowflakeDialer struct {
	// BrokerURL is the optional broker URL. If not specified,
	// we will be using a sensible default value.
	BrokerURL string

	// FrontDomain is the domain to use for fronting. If not
	// specified, we will be using a sensible default.
	FrontDomain string

	// ICEAddresses contains the addresses to use for ICE. If not
	// specified, we will be using a sensible default.
	ICEAddresses []string

	// MaxSnowflakes is the maximum number of snowflakes we
	// should create per dialer. If negative or zero, we will
	// be using a sensible default.
	MaxSnowflakes int

	// newClientTransport is an optional hook for creating
	// an alternative snowflakeTransport in testing.
	newClientTransport func(brokerURL string, frontDomain string,
		iceAddresses []string, keepLocalAddresses bool,
		maxSnowflakes int) (snowflakeTransport, error)
}

// snowflakeTransport is anything that allows us to dial a snowflake
type snowflakeTransport interface {
	Dial() (net.Conn, error)
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
	txp, err := d.newSnowflakeClient(
		d.brokerURL(), d.frontDomain(), d.iceAddresses(),
		false, d.maxSnowflakes(),
	)
	if err != nil {
		return nil, nil, err
	}
	connch, errch := make(chan net.Conn), make(chan error, 1)
	go func() {
		defer close(done) // allow tests to synchronize with this goroutine's exit
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
func (d *SnowflakeDialer) newSnowflakeClient(brokerURL string, frontDomain string,
	iceAddresses []string, keepLocalAddresses bool,
	maxSnowflakes int) (snowflakeTransport, error) {
	if d.newClientTransport != nil {
		return d.newClientTransport(brokerURL, frontDomain, iceAddresses,
			keepLocalAddresses, maxSnowflakes)
	}
	return sflib.NewSnowflakeClient(
		brokerURL, frontDomain, iceAddresses,
		keepLocalAddresses, maxSnowflakes)
}

// brokerURL returns a suitable broker URL.
func (d *SnowflakeDialer) brokerURL() string {
	if d.BrokerURL != "" {
		return d.BrokerURL
	}
	return "https://snowflake-broker.torproject.net.global.prod.fastly.net/"
}

// frontDomain returns a suitable front domain.
func (d *SnowflakeDialer) frontDomain() string {
	if d.FrontDomain != "" {
		return d.FrontDomain
	}
	return "cdn.sstatic.net"
}

// iceAddresses returns suitable ICE addresses.
func (d *SnowflakeDialer) iceAddresses() []string {
	if len(d.ICEAddresses) > 0 {
		return d.ICEAddresses
	}
	return stuninput.AsSnowflakeInput()
}

// maxSnowflakes returns the number of snowflakes to collect.
func (d *SnowflakeDialer) maxSnowflakes() int {
	if d.MaxSnowflakes > 0 {
		return d.MaxSnowflakes
	}
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
