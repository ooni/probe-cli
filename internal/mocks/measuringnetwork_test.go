package mocks

import (
	"testing"

	"github.com/ooni/probe-cli/v3/internal/model"
	utls "gitlab.com/yawning/utls.git"
)

func TestMeasuringN(t *testing.T) {
	t.Run("MockNewDialerWithoutResolver", func(t *testing.T) {
		expected := &Dialer{}
		mn := &MeasuringNetwork{
			MockNewDialerWithoutResolver: func(dl model.DebugLogger, w ...model.DialerWrapper) model.Dialer {
				return expected
			},
		}
		got := mn.NewDialerWithoutResolver(nil, nil)
		if expected != got {
			t.Fatal("unexpected result")
		}
	})

	t.Run("MockNewParallelDNSOverHTTPSResolver", func(t *testing.T) {
		expected := &Resolver{}
		mn := &MeasuringNetwork{
			MockNewParallelDNSOverHTTPSResolver: func(logger model.DebugLogger, URL string) model.Resolver {
				return expected
			},
		}
		got := mn.NewParallelDNSOverHTTPSResolver(nil, "")
		if expected != got {
			t.Fatal("unexpected result")
		}
	})

	t.Run("MockNewParallelUDPResolver", func(t *testing.T) {
		expected := &Resolver{}
		mn := &MeasuringNetwork{
			MockNewParallelUDPResolver: func(logger model.DebugLogger, dialer model.Dialer, address string) model.Resolver {
				return expected
			},
		}
		got := mn.NewParallelUDPResolver(nil, nil, "")
		if expected != got {
			t.Fatal("unexpected result")
		}
	})

	t.Run("MockNewQUICDialerWithoutResolver", func(t *testing.T) {
		expected := &QUICDialer{}
		mn := &MeasuringNetwork{
			MockNewQUICDialerWithoutResolver: func(listener model.UDPListener, logger model.DebugLogger, w ...model.QUICDialerWrapper) model.QUICDialer {
				return expected
			},
		}
		got := mn.NewQUICDialerWithoutResolver(nil, nil, nil)
		if expected != got {
			t.Fatal("unexpected result")
		}
	})

	t.Run("MockNewStdlibResolver", func(t *testing.T) {
		expected := &Resolver{}
		mn := &MeasuringNetwork{
			MockNewStdlibResolver: func(logger model.DebugLogger) model.Resolver {
				return expected
			},
		}
		got := mn.NewStdlibResolver(nil)
		if expected != got {
			t.Fatal("unexpected result")
		}
	})

	t.Run("MockNewTLSHandshakerStdlib", func(t *testing.T) {
		expected := &TLSHandshaker{}
		mn := &MeasuringNetwork{
			MockNewTLSHandshakerStdlib: func(logger model.DebugLogger) model.TLSHandshaker {
				return expected
			},
		}
		got := mn.NewTLSHandshakerStdlib(nil)
		if expected != got {
			t.Fatal("unexpected result")
		}
	})

	t.Run("MockNewTLSHandshakerUTLS", func(t *testing.T) {
		expected := &TLSHandshaker{}
		mn := &MeasuringNetwork{
			MockNewTLSHandshakerUTLS: func(logger model.DebugLogger, id *utls.ClientHelloID) model.TLSHandshaker {
				return expected
			},
		}
		got := mn.NewTLSHandshakerUTLS(nil, nil)
		if expected != got {
			t.Fatal("unexpected result")
		}
	})

	t.Run("MockNewUDPListener", func(t *testing.T) {
		expected := &UDPListener{}
		mn := &MeasuringNetwork{
			MockNewUDPListener: func() model.UDPListener {
				return expected
			},
		}
		got := mn.NewUDPListener()
		if expected != got {
			t.Fatal("unexpected result")
		}
	})
}
