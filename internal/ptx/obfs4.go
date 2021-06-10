package ptx

import (
	"context"
	"fmt"
	"net"
	"path/filepath"
	"time"

	pt "git.torproject.org/pluggable-transports/goptlib.git"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"gitlab.com/yawning/obfs4.git/transports/base"
	"gitlab.com/yawning/obfs4.git/transports/obfs4"
)

// DefaultTestingOBFS4Bridge is a factory that returns you
// an OBFS4Dialer configured for the bridge we use by default
// when testing. Of course, given the nature of obfs4, it's
// not wise to use this bridge in general. But, feel free to
// use this bridge for integration testing of this code.
func DefaultTestingOBFS4Bridge() *OBFS4Dialer {
	// TODO(bassosimone): this is a public working bridge we have found
	// with @hellais. We should ask @cohosh whether there's some obfs4 bridge
	// dedicated to integration testing that we should use instead.
	return &OBFS4Dialer{
		Address:     "109.105.109.165:10527",
		Cert:        "Bvg/itxeL4TWKLP6N1MaQzSOC6tcRIBv6q57DYAZc3b2AzuM+/TfB7mqTFEfXILCjEwzVA",
		DataDir:     "testdata",
		Fingerprint: "8DFCD8FB3285E855F5A55EDDA35696C743ABFC4E",
		IATMode:     "1",
	}
}

// OBFS4Dialer is a dialer for obfs4. Make sure you fill all
// the fields marked as mandatory before using.
type OBFS4Dialer struct {
	// Address contains the MANDATORY proxy address.
	Address string

	// Cert contains the MANDATORY certificate parameter.
	Cert string

	// DataDir is the MANDATORY directory where to store obfs4 data.
	DataDir string

	// Fingerprint is the MANDATORY bridge fingerprint.
	Fingerprint string

	// IATMode contains the MANDATORY iat-mode parameter.
	IATMode string

	// UnderlyingDialer is the optional underlying dialer to
	// use. If not set, we will use &net.Dialer{}.
	UnderlyingDialer UnderlyingDialer
}

// DialContext establishes a connection with the given obfs4 proxy. The context
// argument allows to interrupt this operation midway.
func (d *OBFS4Dialer) DialContext(ctx context.Context) (net.Conn, error) {
	cd, err := d.newCancellableDialer()
	if err != nil {
		return nil, err
	}
	return cd.dial(ctx, "tcp", d.Address)
}

// newCancellableDialer constructs a new cancellable dialer. This function
// is separate from DialContext for testing purposes.
func (d *OBFS4Dialer) newCancellableDialer() (*obfs4CancellableDialer, error) {
	factory := d.newFactory()
	parsedargs, err := d.parseargs(factory)
	if err != nil {
		return nil, err
	}
	return &obfs4CancellableDialer{
		done:       make(chan interface{}),
		ud:         d.underlyingDialer(), // choose proper dialer
		factory:    factory,
		parsedargs: parsedargs,
	}, nil
}

// newFactory creates an obfs4 factory instance.
func (d *OBFS4Dialer) newFactory() base.ClientFactory {
	o4f := &obfs4.Transport{}
	cf, err := o4f.ClientFactory(filepath.Join(d.DataDir, "obfs4"))
	// the source code for this transport always returns a nil error
	runtimex.PanicOnError(err, "unexpected o4f.ClientFactory failure")
	return cf
}

// parseargs parses the obfs4 arguments.
func (d *OBFS4Dialer) parseargs(factory base.ClientFactory) (interface{}, error) {
	args := &pt.Args{"cert": []string{d.Cert}, "iat-mode": []string{d.IATMode}}
	return factory.ParseArgs(args)
}

// underlyingDialer returns a suitable UnderlyingDialer.
func (d *OBFS4Dialer) underlyingDialer() UnderlyingDialer {
	if d.UnderlyingDialer != nil {
		return d.UnderlyingDialer
	}
	return &net.Dialer{
		Timeout: 15 * time.Second, // eventually interrupt connect
	}
}

// obfs4CancellableDialer is a cancellable dialer for obfs4. It will run
// the dial proper in a background goroutine, thus allowing for its early
// cancellation.
type obfs4CancellableDialer struct {
	// done is a channel that will be closed when done. In normal
	// usage you don't want to await for this signal. But it's useful
	// for testing to know that the background goroutine joined.
	done chan interface{}

	// factory is the factory for obfs4.
	factory base.ClientFactory

	// parsedargs contains the parsed args for obfs4.
	parsedargs interface{}

	// ud is the underlying Dialer to use.
	ud UnderlyingDialer
}

// dial performs the dial.
func (d *obfs4CancellableDialer) dial(ctx context.Context, network, address string) (net.Conn, error) {
	connch, errch := make(chan net.Conn), make(chan error, 1)
	go func() {
		defer close(d.done) // signal we're joining
		conn, err := d.factory.Dial(network, address, d.innerDial, d.parsedargs)
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
	case err := <-errch:
		return nil, err
	case conn := <-connch:
		return conn, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// innerDial performs the inner dial using the underlying dialer.
func (d *obfs4CancellableDialer) innerDial(network, address string) (net.Conn, error) {
	return d.ud.DialContext(context.Background(), network, address)
}

// AsBridgeArgument returns the argument to be passed to
// the tor command line to declare this bridge.
func (d *OBFS4Dialer) AsBridgeArgument() string {
	return fmt.Sprintf("obfs4 %s %s cert=%s iat-mode=%s",
		d.Address, d.Fingerprint, d.Cert, d.IATMode)
}

// Name returns the pluggable transport name.
func (d *OBFS4Dialer) Name() string {
	return "obfs4"
}
