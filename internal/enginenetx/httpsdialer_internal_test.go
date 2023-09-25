package enginenetx

import (
	"crypto/tls"
	"errors"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func TestHTTPSDialerVerifyCertificateChain(t *testing.T) {
	t.Run("without any peer certificate", func(t *testing.T) {
		tlsConn := &mocks.TLSConn{
			MockConnectionState: func() tls.ConnectionState {
				return tls.ConnectionState{} // empty!
			},
		}
		certPool := netxlite.NewMozillaCertPool()
		err := httpsDialerVerifyCertificateChain("www.example.com", tlsConn, certPool)
		if !errors.Is(err, errNoPeerCertificate) {
			t.Fatal("unexpected error", err)
		}
	})

	t.Run("with an empty hostname", func(t *testing.T) {
		tlsConn := &mocks.TLSConn{
			MockConnectionState: func() tls.ConnectionState {
				return tls.ConnectionState{} // empty but should not be an issue
			},
		}
		certPool := netxlite.NewMozillaCertPool()
		err := httpsDialerVerifyCertificateChain("", tlsConn, certPool)
		if !errors.Is(err, errEmptyVerifyHostname) {
			t.Fatal("unexpected error", err)
		}
	})
}

func TestHTTPSDialerReduceResult(t *testing.T) {
	t.Run("we return the first conn in a list of conns and close the other conns", func(t *testing.T) {
		var closed int
		expect := &mocks.TLSConn{} // empty
		connv := []model.TLSConn{
			expect,
			&mocks.TLSConn{
				Conn: mocks.Conn{
					MockClose: func() error {
						closed++
						return nil
					},
				},
			},
			&mocks.TLSConn{
				Conn: mocks.Conn{
					MockClose: func() error {
						closed++
						return nil
					},
				},
			},
		}

		conn, err := httpsDialerReduceResult(connv, nil)
		if err != nil {
			t.Fatal(err)
		}

		if conn != expect {
			t.Fatal("unexpected conn")
		}

		if closed != 2 {
			t.Fatal("did not call close")
		}
	})

	t.Run("we join together a list of errors", func(t *testing.T) {
		expectErr := "connection_refused\ninterrupted"
		errorv := []error{errors.New("connection_refused\ninterrupted")}

		conn, err := httpsDialerReduceResult(nil, errorv)
		if err == nil || err.Error() != expectErr {
			t.Fatal("unexpected err", err)
		}

		if conn != nil {
			t.Fatal("expected nil conn")
		}
	})

	t.Run("with a single error we return such an error", func(t *testing.T) {
		expected := errors.New("connection_refused")
		errorv := []error{expected}

		conn, err := httpsDialerReduceResult(nil, errorv)
		if !errors.Is(err, expected) {
			t.Fatal("unexpected err", err)
		}

		if conn != nil {
			t.Fatal("expected nil conn")
		}
	})

	t.Run("we return errDNSNoAnswer if we don't any conns or errors to return", func(t *testing.T) {
		conn, err := httpsDialerReduceResult(nil, nil)
		if !errors.Is(err, errDNSNoAnswer) {
			t.Fatal("unexpected error", err)
		}

		if conn != nil {
			t.Fatal("expected nil conn")
		}
	})
}
