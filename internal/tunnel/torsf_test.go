package tunnel

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/atomicx"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
	"github.com/ooni/probe-cli/v3/internal/ptx"
)

type torsfPTXListenerWrapper struct {
	torsfPTXListener
	counter *atomicx.Int64
}

func (tw *torsfPTXListenerWrapper) Stop() {
	tw.counter.Add(1)
	tw.torsfPTXListener.Stop()
}

func Test_torsfStart(t *testing.T) {
	t.Run("newSnowflakeDialer fails", func(t *testing.T) {
		ctx := context.Background()
		config := &Config{
			Name:                 "torsf",
			Session:              &MockableSession{},
			SnowflakeRendezvous:  "antani", // should cause failure
			TunnelDir:            filepath.Join(os.TempDir(), "torsf-xx"),
			Logger:               model.DiscardLogger,
			TorArgs:              []string{},
			TorBinary:            "",
			testExecabsLookPath:  nil,
			testMkdirAll:         nil,
			testNetListen:        nil,
			testSocks5New:        nil,
			testTorStart:         nil,
			testTorProtocolInfo:  nil,
			testTorEnableNetwork: nil,
			testTorGetInfo:       nil,
		}
		expectDebugInfo := DebugInfo{}
		tun, debugInfo, err := torsfStart(ctx, config)
		if !errors.Is(err, ptx.ErrSnowflakeNoSuchRendezvousMethod) {
			t.Fatal("unexpected err", err)
		}
		if tun != nil {
			t.Fatal("expected nil tun")
		}
		if diff := cmp.Diff(expectDebugInfo, debugInfo); diff != "" {
			t.Fatal(diff)
		}
	})

	t.Run("ptl.Start fails", func(t *testing.T) {
		ctx := context.Background()
		expected := errors.New("mocked error")
		config := &Config{
			Name:                "torsf",
			Session:             &MockableSession{},
			SnowflakeRendezvous: "", // is the default
			TunnelDir:           filepath.Join(os.TempDir(), "torsf-xx"),
			Logger:              model.DiscardLogger,
			TorArgs:             []string{},
			TorBinary:           "",
			testExecabsLookPath: nil,
			testMkdirAll:        nil,
			testNetListen:       nil,
			testSfListenSocks: func(network, laddr string) (ptx.SocksListener, error) {
				return nil, expected
			},
			testSocks5New:        nil,
			testTorStart:         nil,
			testTorProtocolInfo:  nil,
			testTorEnableNetwork: nil,
			testTorGetInfo:       nil,
		}
		expectDebugInfo := DebugInfo{}
		tun, debugInfo, err := torsfStart(ctx, config)
		if !errors.Is(err, expected) {
			t.Fatal("unexpected err", err)
		}
		if tun != nil {
			t.Fatal("expected nil tun")
		}
		if diff := cmp.Diff(expectDebugInfo, debugInfo); diff != "" {
			t.Fatal(diff)
		}
	})

	t.Run("torStart fails", func(t *testing.T) {
		ctx := context.Background()
		ctx, cancel := context.WithCancel(ctx)
		cancel() // should fail immediately
		stopCounter := &atomicx.Int64{}
		config := &Config{
			Name:                "torsf",
			Session:             &MockableSession{},
			SnowflakeRendezvous: "", // is the default
			TunnelDir:           filepath.Join(os.TempDir(), "torsf-xx"),
			Logger:              model.DiscardLogger,
			TorArgs:             []string{},
			TorBinary:           "",
			testExecabsLookPath: nil,
			testMkdirAll:        nil,
			testNetListen:       nil,
			testSfListenSocks:   nil,
			testSfWrapPTXListener: func(tp torsfPTXListener) torsfPTXListener {
				return &torsfPTXListenerWrapper{
					torsfPTXListener: tp,
					counter:          stopCounter,
				}
			},
			testSocks5New:        nil,
			testTorStart:         nil,
			testTorProtocolInfo:  nil,
			testTorEnableNetwork: nil,
			testTorGetInfo:       nil,
		}
		expectDebugInfo := DebugInfo{
			Name: "torsf",
		}
		tun, debugInfo, err := torsfStart(ctx, config)
		if !errors.Is(err, context.Canceled) {
			t.Fatal("unexpected err", err)
		}
		if tun != nil {
			t.Fatal("expected nil tun")
		}
		if diff := cmp.Diff(expectDebugInfo, debugInfo); diff != "" {
			t.Fatal(diff)
		}
		if stopCounter.Load() != 1 {
			t.Fatal("did not call stop")
		}
	})

	t.Run("on success", func(t *testing.T) {
		ctx := context.Background()
		expectDebugInfo := DebugInfo{
			Name: "torsf",
		}
		config := &Config{
			Name:                "torsf",
			Session:             &MockableSession{},
			SnowflakeRendezvous: "", // is the default
			TunnelDir:           filepath.Join(os.TempDir(), "torsf-xx"),
			Logger:              model.DiscardLogger,
			TorArgs:             []string{},
			TorBinary:           "",
			testExecabsLookPath: nil,
			testMkdirAll:        nil,
			testNetListen:       nil,
			testSfListenSocks:   nil,
			testSfTorStart: func(ctx context.Context, config *Config) (Tunnel, DebugInfo, error) {
				tun := &fakeTunnel{
					addr: &mocks.Addr{
						MockString: func() string {
							return "127.0.0.1:5555"
						},
					},
					bootstrapTime: 123,
					listener: &mocks.Listener{
						MockClose: func() error {
							return nil
						},
					},
					once: sync.Once{},
				}
				return tun, expectDebugInfo, nil
			},
			testSocks5New:        nil,
			testTorStart:         nil,
			testTorProtocolInfo:  nil,
			testTorEnableNetwork: nil,
			testTorGetInfo:       nil,
		}
		tun, debugInfo, err := torsfStart(ctx, config)
		if err != nil {
			t.Fatal(err)
		}
		if diff := cmp.Diff(expectDebugInfo, debugInfo); diff != "" {
			t.Fatal(diff)
		}
		if tun.BootstrapTime() != 123 {
			t.Fatal("invalid bootstrap time")
		}
		if tun.SOCKS5ProxyURL().String() != "socks5://127.0.0.1:5555" {
			t.Fatal("invalid socks5 proxy URL")
		}
		tun.Stop()
	})
}
