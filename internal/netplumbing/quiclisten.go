package netplumbing

import (
	"context"
	"net"
)

// QUICListener is a listener for QUIC.
type QUICListener interface {
	// QUICListen starts a listening UDP connection for QUIC.
	QUICListen(ctx context.Context) (*net.UDPConn, error)
}

// quicStdlibListener is a QUICListener using the Go stdlib.
type quicStdlibListener struct{}

// QUICListen implements QUICListener.QUICListen.
func (ql *quicStdlibListener) QUICListen(ctx context.Context) (*net.UDPConn, error) {
	return net.ListenUDP("udp", &net.UDPAddr{})
}

// DefaultQUICListener returns the default QUICListener.
func (txp *Transport) DefaultQUICListener() QUICListener {
	return &quicStdlibListener{}
}

// ErrListen is a listen error.
type ErrListen struct {
	error
}

// Unwrap returns the underlying error.
func (err *ErrListen) Unwrap() error {
	return err.error
}

// quicListen creates a new listening UDP connection for QUIC.
func (txp *Transport) quicListen(ctx context.Context) (net.PacketConn, error) {
	ql := txp.DefaultQUICListener()
	if config := ContextConfig(ctx); config != nil && config.QUICListener != nil {
		ql = config.QUICListener
	}
	log := txp.logger(ctx)
	log.Debug("quic: start listening...")
	conn, err := ql.QUICListen(ctx)
	if err != nil {
		log.Debugf("quic: start listening... %s", err)
		return nil, &ErrListen{err}
	}
	log.Debugf("quic: start listening... %s", conn.LocalAddr().String())
	return &quicUDPConnWrapper{
		byteCounter: txp.byteCounter(ctx), UDPConn: conn}, nil
}
