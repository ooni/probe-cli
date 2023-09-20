package enginenetx

import (
	"crypto/tls"
	"errors"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/mocks"
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
}
