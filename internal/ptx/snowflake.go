package ptx

import (
	"context"
	"net"

	sflib "git.torproject.org/pluggable-transports/snowflake.git/v2/client/lib"
	"github.com/ooni/probe-cli/v3/internal/stuninput"
)

// SnowflakeDialer is a dialer for snowflake. When optional fields are
// not specified, we use defaults from the snowflake repository.
type SnowflakeDialer struct {
	// newClientTransport is an optional hook for creating
	// an alternative snowflakeTransport in testing.
	newClientTransport func(config sflib.ClientConfig) (snowflakeTransport, error)
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
	txp, err := d.newSnowflakeClient(sflib.ClientConfig{
		BrokerURL:          d.brokerURL(),
		AmpCacheURL:        d.ampCacheURL(),
		FrontDomain:        d.frontDomain(),
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
func (d *SnowflakeDialer) newSnowflakeClient(config sflib.ClientConfig) (snowflakeTransport, error) {
	if d.newClientTransport != nil {
		return d.newClientTransport(config)
	}
	return sflib.NewSnowflakeClient(config)
}

// ampCacheURL returns a suitable AMP cache URL.
func (d *SnowflakeDialer) ampCacheURL() string {
	// I tried using the following AMP cache and always got:
	//
	// 2022/01/19 16:51:28 AMP cache rendezvous response: 500 Internal Server Error
	//
	// So I disabled the AMP cache until we figure it out.
	//
	//return "https://cdn.ampproject.org/"
	return ""
}

// brokerURL returns a suitable broker URL.
func (d *SnowflakeDialer) brokerURL() string {
	return "https://snowflake-broker.torproject.net.global.prod.fastly.net/"
}

// frontDomain returns a suitable front domain.
func (d *SnowflakeDialer) frontDomain() string {
	return "cdn.sstatic.net"
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
