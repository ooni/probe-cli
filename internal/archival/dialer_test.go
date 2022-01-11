package archival

import (
	"context"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func TestSaverDialContext(t *testing.T) {
	// newConn creates a new connection with the desired properties.
	newConn := func(address string) net.Conn {
		return &mocks.Conn{
			MockRemoteAddr: func() net.Addr {
				return &mocks.Addr{
					MockString: func() string {
						return address
					},
				}
			},
			MockClose: func() error {
				return nil
			},
		}
	}

	// newDialer creates a dialer for testing.
	newDialer := func(conn net.Conn, err error) model.Dialer {
		return &mocks.Dialer{
			MockDialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
				time.Sleep(1 * time.Microsecond)
				return conn, err
			},
		}
	}

	t.Run("on success", func(t *testing.T) {
		const mockedEndpoint = "8.8.4.4:443"
		dialer := newDialer(newConn(mockedEndpoint), nil)
		saver := NewSaver()
		v := &SingleNetworkEventValidator{
			ExpectedCount:   0,
			ExpectedErr:     nil,
			ExpectedNetwork: "tcp",
			ExpectedOp:      netxlite.ConnectOperation,
			ExpectedEpnt:    mockedEndpoint,
			Saver:           saver,
		}
		ctx := context.Background()
		conn, err := saver.DialContext(ctx, dialer, "tcp", mockedEndpoint)
		if err != nil {
			t.Fatal(err)
		}
		if conn == nil {
			t.Fatal("expected non-nil conn")
		}
		conn.Close()
		if err := v.Validate(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("on failure", func(t *testing.T) {
		const mockedEndpoint = "8.8.4.4:443"
		mockedError := netxlite.NewTopLevelGenericErrWrapper(io.EOF)
		dialer := newDialer(nil, mockedError)
		saver := NewSaver()
		v := &SingleNetworkEventValidator{
			ExpectedCount:   0,
			ExpectedErr:     mockedError,
			ExpectedNetwork: "tcp",
			ExpectedOp:      netxlite.ConnectOperation,
			ExpectedEpnt:    mockedEndpoint,
			Saver:           saver,
		}
		ctx := context.Background()
		conn, err := saver.DialContext(ctx, dialer, "tcp", mockedEndpoint)
		if !errors.Is(err, mockedError) {
			t.Fatal("unexpected err", err)
		}
		if conn != nil {
			t.Fatal("expected nil conn")
		}
		if err := v.Validate(); err != nil {
			t.Fatal(err)
		}
	})
}

func TestSaverRead(t *testing.T) {
	// newConn is a helper function for creating a new connection.
	newConn := func(endpoint string, numBytes int, err error) net.Conn {
		return &mocks.Conn{
			MockRead: func(b []byte) (int, error) {
				time.Sleep(time.Microsecond)
				return numBytes, err
			},
			MockRemoteAddr: func() net.Addr {
				return &mocks.Addr{
					MockString: func() string {
						return endpoint
					},
					MockNetwork: func() string {
						return "tcp"
					},
				}
			},
		}
	}

	t.Run("on success", func(t *testing.T) {
		const mockedEndpoint = "8.8.4.4:443"
		const mockedNumBytes = 128
		conn := newConn(mockedEndpoint, mockedNumBytes, nil)
		saver := NewSaver()
		v := &SingleNetworkEventValidator{
			ExpectedCount:   mockedNumBytes,
			ExpectedErr:     nil,
			ExpectedNetwork: "tcp",
			ExpectedOp:      netxlite.ReadOperation,
			ExpectedEpnt:    mockedEndpoint,
			Saver:           saver,
		}
		buf := make([]byte, 1024)
		count, err := saver.Read(conn, buf)
		if err != nil {
			t.Fatal(err)
		}
		if count != mockedNumBytes {
			t.Fatal("unexpected count")
		}
		if err := v.Validate(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("on failure", func(t *testing.T) {
		const mockedEndpoint = "8.8.4.4:443"
		mockedError := netxlite.NewTopLevelGenericErrWrapper(io.EOF)
		conn := newConn(mockedEndpoint, 0, mockedError)
		saver := NewSaver()
		v := &SingleNetworkEventValidator{
			ExpectedCount:   0,
			ExpectedErr:     mockedError,
			ExpectedNetwork: "tcp",
			ExpectedOp:      netxlite.ReadOperation,
			ExpectedEpnt:    mockedEndpoint,
			Saver:           saver,
		}
		buf := make([]byte, 1024)
		count, err := saver.Read(conn, buf)
		if !errors.Is(err, mockedError) {
			t.Fatal("unexpected err", err)
		}
		if count != 0 {
			t.Fatal("unexpected count")
		}
		if err := v.Validate(); err != nil {
			t.Fatal(err)
		}
	})
}

func TestSaverWrite(t *testing.T) {
	// newConn is a helper function for creating a new connection.
	newConn := func(endpoint string, numBytes int, err error) net.Conn {
		return &mocks.Conn{
			MockWrite: func(b []byte) (int, error) {
				time.Sleep(time.Microsecond)
				return numBytes, err
			},
			MockRemoteAddr: func() net.Addr {
				return &mocks.Addr{
					MockString: func() string {
						return endpoint
					},
					MockNetwork: func() string {
						return "tcp"
					},
				}
			},
		}
	}

	t.Run("on success", func(t *testing.T) {
		const mockedEndpoint = "8.8.4.4:443"
		const mockedNumBytes = 128
		conn := newConn(mockedEndpoint, mockedNumBytes, nil)
		saver := NewSaver()
		v := &SingleNetworkEventValidator{
			ExpectedCount:   mockedNumBytes,
			ExpectedErr:     nil,
			ExpectedNetwork: "tcp",
			ExpectedOp:      netxlite.WriteOperation,
			ExpectedEpnt:    mockedEndpoint,
			Saver:           saver,
		}
		buf := make([]byte, 1024)
		count, err := saver.Write(conn, buf)
		if err != nil {
			t.Fatal(err)
		}
		if count != mockedNumBytes {
			t.Fatal("unexpected count")
		}
		if err := v.Validate(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("on failure", func(t *testing.T) {
		const mockedEndpoint = "8.8.4.4:443"
		mockedError := netxlite.NewTopLevelGenericErrWrapper(io.EOF)
		conn := newConn(mockedEndpoint, 0, mockedError)
		saver := NewSaver()
		v := &SingleNetworkEventValidator{
			ExpectedCount:   0,
			ExpectedErr:     mockedError,
			ExpectedNetwork: "tcp",
			ExpectedOp:      netxlite.WriteOperation,
			ExpectedEpnt:    mockedEndpoint,
			Saver:           saver,
		}
		buf := make([]byte, 1024)
		count, err := saver.Write(conn, buf)
		if !errors.Is(err, mockedError) {
			t.Fatal("unexpected err", err)
		}
		if count != 0 {
			t.Fatal("unexpected count")
		}
		if err := v.Validate(); err != nil {
			t.Fatal(err)
		}
	})
}

// SingleNetworkEventValidator expects to find a single
// network event inside of the saver and ensures that such
// an event contains the required field values.
type SingleNetworkEventValidator struct {
	ExpectedCount   int
	ExpectedErr     error
	ExpectedNetwork string
	ExpectedOp      string
	ExpectedEpnt    string
	Saver           *Saver
}

func (v *SingleNetworkEventValidator) Validate() error {
	trace := v.Saver.MoveOutTrace()
	if len(trace.Network) != 1 {
		return errors.New("expected to see a single .Network event")
	}
	entry := trace.Network[0]
	if entry.Count != v.ExpectedCount {
		return errors.New("expected to see a different .Count")
	}
	if !errors.Is(entry.Failure, v.ExpectedErr) {
		return errors.New("unexpected .Failure")
	}
	if !entry.Finished.After(entry.Started) {
		return errors.New(".Finished should be after .Started")
	}
	if entry.Network != v.ExpectedNetwork {
		return errors.New("invalid value for .Network")
	}
	if entry.Operation != v.ExpectedOp {
		return errors.New("invalid value for .Operation")
	}
	if entry.RemoteAddr != v.ExpectedEpnt {
		return errors.New("unexpected value for .RemoteAddr")
	}
	return nil
}
