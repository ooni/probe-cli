package remote

import (
	"crypto/x509"
	"net"
	"time"

	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// UnderlyingNetworkConfig contains configuration for [NewUnderlyingNetwork].
//
// The zero value of this struct is invalid; init all MANDATORY fields.
type UnderlyingNetworkConfig struct {
	// Conn is the MANDATORY net.Conn to use as transport. [NewUnderlyingNetwork] BORROWS
	// this net.Conn for sending and receiving packets through it.
	Conn net.Conn

	// LocalAddress is the MANDATORY local address.
	LocalAddress string

	// Logger is the MANDATORY logger to use.
	Logger model.Logger

	// MTU is the OPTIONAL MTU to use.
	MTU uint32

	// ResolverAddress is the OPTIONAL default resolver address.
	ResolverAddress string

	// RemoteAddress is the MANDATORY remote address.
	RemoteAddress string
}

// NewUnderlyingNetwork creates a new [model.UnderlyingNetwork] that
// sends and receives packets over the configured [net.Conn].
//
// This function creates background goroutines for sending and receiving
// IP packets over the configured [net.Conn]. These goroutines will run
// until either the configured [net.Conn] or the returned underlying network
// are open. Thus, your cleanup code should defer close both of them.
func NewUnderlyingNetwork(config *UnderlyingNetworkConfig) (model.UnderlyingNetwork, error) {
	// Implementation note: a netem.UNetStack is designed to work with a custom
	// TLS certificate pool to allow impersonating servers when doing QA.
	//
	// However, we can wrap the underlying network returned to the caller with
	// an wrapper routing the DefaultCertPool method back to the CA we would be
	// using if we were not using netem under the hood.

	cfg, err := netem.NewTLSMITMConfig()
	if err != nil {
		return nil, err
	}

	// you MUST use at least 1252 bytes if you want to use github.com/quic-go/quic-go
	MTU := config.MTU
	if MTU <= 0 {
		MTU = 1500
	}

	resoAddr := config.ResolverAddress
	if resoAddr == "" {
		resoAddr = "8.8.4.4"
	}

	stack, err := netem.NewUNetStack(config.Logger, MTU, config.LocalAddress, cfg, resoAddr)
	if err != nil {
		return nil, err
	}

	unet := &underlyingNetworkOverrideDefaultCertPool{
		&netxlite.NetemUnderlyingNetworkAdapter{UNet: stack}}

	// Up until this point, we have an userspace TCP/IP stack that is not attached to
	// any network interface. We need to attach it to the config.Conn, such that we
	// will be able to route packets using the conn.
	go underlyingNetworkReadLoop(config.Logger, config.Conn, stack)
	go underlyingNetworkWriteLoop(config.Logger, config.Conn, stack)

	return unet, nil
}

var defaultTProxy = &netxlite.DefaultTProxy{}

type underlyingNetworkOverrideDefaultCertPool struct {
	model.UnderlyingNetwork
}

func (unw *underlyingNetworkOverrideDefaultCertPool) DefaultCertPool() *x509.CertPool {
	return defaultTProxy.DefaultCertPool()
}

func underlyingNetworkReadLoop(logger model.Logger, conn net.Conn, stack *netem.UNetStack) {
	defer logger.Infof("underlyingNetworkReadLoop: done")
	logger.Infof("underlyingNetworkReadLoop: start")
	for {
		ipPacket, err := ReadPacket(conn)
		if err != nil {
			logger.Warnf("underlyingNetworkReadLoop: ReadPacket: %s", err.Error())
			return
		}
		frame := &netem.Frame{
			Deadline: time.Time{}, // irrelevant in this context
			Flags:    0,           // irrelevant in this context
			Payload:  ipPacket,
			Spoofed:  [][]byte{}, // irrelevant in this context
		}
		if err := stack.WriteFrame(frame); err != nil {
			logger.Warnf("underlyingNetworkReadLoop: stack.WriteFrame: %s", err.Error())
			return
		}
	}
}

func underlyingNetworkWriteLoop(logger model.Logger, conn net.Conn, stack *netem.UNetStack) {
	defer logger.Infof("underlyingNetworkWritedLoop: done")
	logger.Infof("underlyingNetworkWritedLoop: start")
	for {
		select {
		case <-stack.StackClosed():
			logger.Warnf("underlyingNetworkWriteLoop: stack closed")
			return
		case <-stack.FrameAvailable():
			frame, err := stack.ReadFrameNonblocking()
			if err != nil {
				logger.Warnf("underlyingNetworkWriteLoop: stack.ReadFrameNonblocking: %s", err.Error())
				return
			}
			ipPacket := frame.Payload
			if err := WritePacket(conn, ipPacket); err != nil {
				logger.Warnf("underlyingNetworkReadLoop: WritePacket: %s", err.Error())
				return
			}
		}
	}
}
