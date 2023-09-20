package enginenetx

import (
	"context"
	"crypto/tls"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func TestHTTPSDialerTacticsEmitter(t *testing.T) {
	t.Run("we correctly handle the case of a canceled context", func(t *testing.T) {
		hd := &HTTPSDialer{
			idGenerator: &atomic.Int64{},
			logger:      model.DiscardLogger,
			policy:      &HTTPSDialerNullPolicy{},
			resolver:    netxlite.NewStdlibResolver(model.DiscardLogger),
			rootCAs:     netxlite.NewMozillaCertPool(),
			unet:        &netxlite.DefaultTProxy{},
			wg:          &sync.WaitGroup{},
		}

		tactics := []HTTPSDialerTactic{
			&httpsDialerNullTactic{
				Address: "10.0.0.1",
				Delay:   0,
				Domain:  "www.example.com",
			},
			&httpsDialerNullTactic{
				Address: "10.0.0.2",
				Delay:   0,
				Domain:  "www.example.com",
			},
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // we want the tested function to run with a canceled context

		out := hd.tacticsEmitter(ctx, tactics...)

		var count int
		for range out {
			count++
		}

		if count != 0 {
			t.Fatal("nothing should have been emitted here")
		}
	})
}

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
