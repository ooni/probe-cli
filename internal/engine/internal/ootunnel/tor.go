package ootunnel

import (
	"context"
	"net/url"
	"strings"
	"time"

	"github.com/cretz/bine/control"
	"github.com/cretz/bine/tor"
)

// torLibrary is the library for managing tor.
type torLibrary interface {
	// Start starts the tor instance.
	Start(ctx context.Context, conf *tor.StartConf) (torInstance, error)
}

// cretzBineTorLibrary is the default torLibrary.
type cretzBineTorLibrary struct{}

// Start implements torLibrary.Start.
func (*cretzBineTorLibrary) Start(ctx context.Context, conf *tor.StartConf) (torInstance, error) {
	instance, err := tor.Start(ctx, conf)
	if err != nil {
		return nil, err
	}
	return &cretzBineTorInstance{instance}, nil
}

// torInstance is a running instance of tor.
type torInstance interface {
	// SetStopProcessOnClose sets the value of
	// the StopProcessOnClose property.
	SetStopProcessOnClose(value bool)

	// EnableNetwork enables or disables the network.
	EnableNetwork(ctx context.Context, wait bool) error

	// ControlConn returns the control connection.
	ControlConn() torControlConn

	// Close closes the instance.
	Close() error
}

// cretzBineTorInstance is the default torInstance.
type cretzBineTorInstance struct {
	instance *tor.Tor
}

// SetStopProcessOnClose implements torInstance.SetStopProcessOnClose.
func (ti *cretzBineTorInstance) SetStopProcessOnClose(value bool) {
	ti.instance.StopProcessOnClose = value
}

// EnableNetwork implements torInstance.EnableNetwork.
func (ti *cretzBineTorInstance) EnableNetwork(ctx context.Context, wait bool) error {
	return ti.instance.EnableNetwork(ctx, wait)
}

// ControlConn implements torInstance.ControlConn.
func (ti *cretzBineTorInstance) ControlConn() torControlConn {
	return &cretzBineControlConn{ti.instance.Control}
}

// Close implements torInstance.Close.
func (ti *cretzBineTorInstance) Close() error {
	return ti.instance.Close()
}

// torControlConn is the control connection.
type torControlConn interface {
	// GetInfo returns information on specific keys.
	GetInfo(keys ...string) ([]*control.KeyVal, error)
}

// cretzBineControlConn is the default torControlConn.
type cretzBineControlConn struct {
	conn *control.Conn
}

// GetInfo implements torControlConn.GetInfo.
func (ti *cretzBineControlConn) GetInfo(keys ...string) ([]*control.KeyVal, error) {
	return ti.conn.GetInfo(keys...)
}

// getTorLibrary returns the torLibrary we're using.
func (b *Broker) getTorLibrary() torLibrary {
	if b.torLibrary != nil {
		return b.torLibrary
	}
	return &cretzBineTorLibrary{}
}

// newTor starts a tor tunnel.
func (b *Broker) newTor(ctx context.Context, config *Config) (Tunnel, error) {
	extraArgs := append([]string{}, config.TorArgs...)
	extraArgs = append(extraArgs, "Log")
	extraArgs = append(extraArgs, "notice stderr")
	instance, err := b.getTorLibrary().Start(ctx, &tor.StartConf{
		DataDir:   config.StateDir,
		ExtraArgs: extraArgs,
		ExePath:   config.TorBinary, // empty means use default
		NoHush:    true,
	})
	if err != nil {
		return nil, err
	}
	instance.SetStopProcessOnClose(true)
	start := time.Now()
	if err := instance.EnableNetwork(ctx, true); err != nil {
		instance.Close()
		return nil, err
	}
	stop := time.Now()
	// Adapted from <https://git.io/Jfc7N>
	cc := instance.ControlConn()
	info, err := cc.GetInfo("net/listeners/socks")
	if err != nil {
		instance.Close()
		return nil, err
	}
	if len(info) != 1 || info[0].Key != "net/listeners/socks" {
		instance.Close()
		return nil, ErrNoSOCKSProxy
	}
	proxyAddress := info[0].Val
	if strings.HasPrefix(proxyAddress, "unix:") {
		instance.Close()
		return nil, ErrUnsupportedProxy
	}
	return &tunnelish{
		bootstrapTime:         stop.Sub(start),
		deleteStateDirOnClose: config.DeleteStateDirOnClose,
		name:                  Tor,
		proxyURL:              &url.URL{Scheme: "socks5", Host: proxyAddress},
		stateDir:              config.StateDir,
		t:                     instance,
	}, nil
}
