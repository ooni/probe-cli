package enginenetx

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
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

		var tactics []HTTPSDialerTactic
		for idx := 0; idx < 255; idx++ {
			tactics = append(tactics, &httpsDialerNullTactic{
				Address: fmt.Sprintf("10.0.0.%d", idx),
				Delay:   0,
				Domain:  "www.example.com",
			})
		}

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // we want the tested function to run with a canceled context

		out := hd.tacticsEmitter(ctx, tactics...)

		for range out {
			// Here we do nothing!
			//
			// Ideally, we would like to count and assert that we have
			// got no tactic from the channel but the selection of ready
			// channels is nondeterministic, so we cannot really be
			// asserting that. This leaves us with asking the question
			// of what we should be asserting here?
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
