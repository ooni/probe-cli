package urlgetter

import (
	"context"
	"fmt"
	"net/url"
	"path/filepath"
	"sync"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine/model"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/archival"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/errorx"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/trace"
	"github.com/ooni/probe-cli/v3/internal/engine/tunnel"
)

// The Getter gets the specified target in the context of the
// given session and with the specified config.
//
// Other OONI experiment should use the Getter to factor code when
// the Getter implements the operations they wanna perform.
type Getter struct {
	// Begin is the time when the experiment begun. If you do not
	// set this field, every target is measured independently.
	Begin time.Time

	// Config contains settings for this run. If not set, then
	// we will use the default config.
	Config Config

	// Session is the session for this run. This field must
	// be set otherwise the code will panic.
	Session model.ExperimentSession

	// Target is the thing to measure in this run. This field must
	// be set otherwise the code won't know what to do.
	Target string
}

// Get performs the action described by g using the given context
// and returning the test keys and eventually an error
func (g Getter) Get(ctx context.Context) (TestKeys, error) {
	if g.Config.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, g.Config.Timeout)
		defer cancel()
	}
	if g.Begin.IsZero() {
		g.Begin = time.Now()
	}
	saver := new(trace.Saver)
	tk, err := g.get(ctx, saver)
	// Make sure we have an operation in cases where we fail before
	// hitting our httptransport that does error wrapping.
	err = errorx.SafeErrWrapperBuilder{
		Error:     err,
		Operation: errorx.TopLevelOperation,
	}.MaybeBuild()
	tk.FailedOperation = archival.NewFailedOperation(err)
	tk.Failure = archival.NewFailure(err)
	events := saver.Read()
	tk.Queries = append(tk.Queries, archival.NewDNSQueriesList(g.Begin, events)...)
	tk.NetworkEvents = append(
		tk.NetworkEvents, archival.NewNetworkEventsList(g.Begin, events)...,
	)
	tk.Requests = append(
		tk.Requests, archival.NewRequestList(g.Begin, events)...,
	)
	if len(tk.Requests) > 0 {
		// OONI's convention is that the last request appears first
		tk.HTTPResponseStatus = tk.Requests[0].Response.Code
		tk.HTTPResponseBody = tk.Requests[0].Response.Body.Value
		tk.HTTPResponseLocations = tk.Requests[0].Response.Locations
	}
	tk.TCPConnect = append(
		tk.TCPConnect, archival.NewTCPConnectList(g.Begin, events)...,
	)
	tk.TLSHandshakes = append(
		tk.TLSHandshakes, archival.NewTLSHandshakesList(g.Begin, events)...,
	)
	return tk, err
}

var (
	// tunnelDirCount counts the number of tunnels started by
	// the urlgetter package so far.
	tunnelDirCount int64

	// tunnelDirMu protects tunnelDirCount
	tunnelDirMu sync.Mutex
)

func (g Getter) get(ctx context.Context, saver *trace.Saver) (TestKeys, error) {
	tk := TestKeys{
		Agent:  "redirect",
		Tunnel: g.Config.Tunnel,
	}
	if g.Config.DNSCache != "" {
		tk.DNSCache = []string{g.Config.DNSCache}
	}
	if g.Config.NoFollowRedirects {
		tk.Agent = "agent"
	}
	// start tunnel
	var proxyURL *url.URL
	if g.Config.Tunnel != "" {
		// Every new instance of the tunnel goes into a separate
		// directory within the temporary directory. Calling
		// Session.Close will delete such a directory.
		tunnelDirMu.Lock()
		count := tunnelDirCount
		tunnelDirCount++
		tunnelDirMu.Unlock()
		tun, err := tunnel.Start(ctx, &tunnel.Config{
			Name:      g.Config.Tunnel,
			Session:   g.Session,
			TorArgs:   g.Session.TorArgs(),
			TorBinary: g.Session.TorBinary(),
			TunnelDir: filepath.Join(
				g.Session.TempDir(), fmt.Sprintf("urlgetter-tunnel-%d", count)),
		})
		if err != nil {
			return tk, err
		}
		tk.BootstrapTime = tun.BootstrapTime().Seconds()
		proxyURL = tun.SOCKS5ProxyURL()
		tk.SOCKSProxy = proxyURL.String()
		defer tun.Stop()
	}
	// create configuration
	configurer := Configurer{
		Config:   g.Config,
		Logger:   g.Session.Logger(),
		ProxyURL: proxyURL,
		Saver:    saver,
	}
	configuration, err := configurer.NewConfiguration()
	if err != nil {
		return tk, err
	}
	defer configuration.CloseIdleConnections()
	// run the measurement
	runner := Runner{
		Config:     g.Config,
		HTTPConfig: configuration.HTTPConfig,
		Target:     g.Target,
	}
	return tk, runner.Run(ctx)
}
