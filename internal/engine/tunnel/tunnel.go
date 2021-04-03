// Package tunnel allows to create tunnels to speak
// with OONI backends and other services.
package tunnel

import (
	"context"
	"errors"
	"net/url"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine/model"
)

// Session is the way in which this package sees a Session.
type Session interface {
	FetchPsiphonConfig(ctx context.Context) ([]byte, error)
	TempDir() string
	TorArgs() []string
	TorBinary() string
	Logger() model.Logger
}

// Tunnel is a tunnel used by the session
type Tunnel interface {
	BootstrapTime() time.Duration
	SOCKS5ProxyURL() *url.URL
	Stop()
}

// Config contains config for the session tunnel.
type Config struct {
	Name    string
	Session Session
	WorkDir string
}

// Start starts a new tunnel by name or returns an error. Note that if you
// pass to this function the "" tunnel, you get back nil, nil.
func Start(ctx context.Context, config Config) (Tunnel, error) {
	logger := config.Session.Logger()
	switch config.Name {
	case "":
		logger.Debugf("no tunnel has been requested")
		return enforceNilContract(nil, nil)
	case "psiphon":
		logger.Infof("starting %s tunnel; please be patient...", config.Name)
		tun, err := psiphonStart(ctx, config.Session, psiphonConfig{
			WorkDir: config.WorkDir,
		})
		return enforceNilContract(tun, err)
	case "tor":
		logger.Infof("starting %s tunnel; please be patient...", config.Name)
		tun, err := torStart(ctx, config.Session)
		return enforceNilContract(tun, err)
	default:
		return nil, errors.New("unsupported tunnel")
	}
}

func enforceNilContract(tun Tunnel, err error) (Tunnel, error) {
	if err != nil {
		return nil, err
	}
	return tun, nil
}
