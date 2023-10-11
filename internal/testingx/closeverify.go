package testingx

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// CloseVerify verifies that we're closing all connections.
//
// The zero value of this struct is ready to use.
type CloseVerify struct {
	mu    sync.Mutex
	conns map[string]io.Closer
}

func (cv *CloseVerify) addConn(key string, closer io.Closer) {
	defer cv.mu.Unlock()
	cv.mu.Lock()
	if cv.conns == nil {
		cv.conns = make(map[string]io.Closer)
	}
	_, good := cv.conns[key]
	runtimex.Assert(!good, fmt.Sprintf("we're already tracking: %s", key))
	cv.conns[key] = closer
}

func (cv *CloseVerify) removeConn(key string) {
	defer cv.mu.Unlock()
	cv.mu.Lock()
	_, good := cv.conns[key]
	runtimex.Assert(good, fmt.Sprintf("we're not tracking: %s", key))
	delete(cv.conns, key)
}

// CheckForOpenConns returns an error if we still have some open connections.
func (cv *CloseVerify) CheckForOpenConns() error {
	defer cv.mu.Unlock()
	cv.mu.Lock()
	var errorv []error
	for key := range cv.conns {
		errorv = append(errorv, fmt.Errorf("%s has not been closed", key))
	}
	return errors.Join(errorv...) // returns nil if empty
}

// WrapUnderlyingNetwork returns a [model.UnderlyingNetwork] that comunicates
// sockets open and close events to the [*CloseVerify] struct.
func (cv *CloseVerify) WrapUnderlyingNetwork(unet model.UnderlyingNetwork) model.UnderlyingNetwork {
	return &closeVerifyUnderlyingNetwork{
		UnderlyingNetwork: unet,
		cv:                cv,
	}
}

type closeVerifyUnderlyingNetwork struct {
	model.UnderlyingNetwork
	cv *CloseVerify
}

// DialContext implements model.UnderlyingNetwork.
func (unet *closeVerifyUnderlyingNetwork) DialContext(
	ctx context.Context, network, address string) (net.Conn, error) {
	conn, err := unet.UnderlyingNetwork.DialContext(ctx, network, address)
	if err != nil {
		return nil, err
	}

	localAddr := conn.LocalAddr()
	key := fmt.Sprintf("%s/%s", localAddr.String(), localAddr.Network())
	conn = &closeVerifyConn{
		Conn: conn,
		cv:   unet.cv,
		key:  key,
		once: sync.Once{},
	}

	unet.cv.addConn(key, conn)

	return conn, nil
}

type closeVerifyConn struct {
	net.Conn
	cv   *CloseVerify
	key  string
	once sync.Once
}

func (c *closeVerifyConn) Close() (err error) {
	c.once.Do(func() {
		c.cv.removeConn(c.key)
		err = c.Conn.Close()
	})
	return
}

// ListenTCP implements model.UnderlyingNetwork.
func (unet *closeVerifyUnderlyingNetwork) ListenTCP(
	network string, addr *net.TCPAddr) (net.Listener, error) {
	listener, err := unet.UnderlyingNetwork.ListenTCP(network, addr)
	if err != nil {
		return nil, err
	}

	localAddr := listener.Addr()
	key := fmt.Sprintf("%s/%s", localAddr.String(), localAddr.Network())
	listener = &closeVerifyListener{
		Listener: listener,
		cv:       unet.cv,
		key:      key,
		once:     sync.Once{},
	}

	unet.cv.addConn(key, listener)

	return listener, nil
}

type closeVerifyListener struct {
	net.Listener
	cv   *CloseVerify
	key  string
	once sync.Once
}

func (c *closeVerifyListener) Accept() (net.Conn, error) {
	conn, err := c.Listener.Accept()
	if err != nil {
		return nil, err
	}

	localAddr := conn.LocalAddr()
	key := fmt.Sprintf("%s/%s", localAddr.String(), localAddr.Network())
	conn = &closeVerifyConn{
		Conn: conn,
		cv:   c.cv,
		key:  key,
		once: sync.Once{},
	}

	c.cv.addConn(key, conn)

	return conn, nil
}

func (c *closeVerifyListener) Close() (err error) {
	c.once.Do(func() {
		c.cv.removeConn(c.key)
		err = c.Listener.Close()
	})
	return
}

func (unet *closeVerifyUnderlyingNetwork) ListenUDP(
	network string, addr *net.UDPAddr) (model.UDPLikeConn, error) {
	pconn, err := unet.UnderlyingNetwork.ListenUDP(network, addr)
	if err != nil {
		return nil, err
	}

	localAddr := pconn.LocalAddr()
	key := fmt.Sprintf("%s/%s", localAddr.String(), localAddr.Network())
	pconn = &closeVerifyUDPConn{
		UDPLikeConn: pconn,
		cv:          unet.cv,
		key:         key,
		once:        sync.Once{},
	}

	unet.cv.addConn(key, pconn)

	return pconn, nil
}

type closeVerifyUDPConn struct {
	model.UDPLikeConn
	cv   *CloseVerify
	key  string
	once sync.Once
}

func (c *closeVerifyUDPConn) Close() (err error) {
	c.once.Do(func() {
		c.cv.removeConn(c.key)
		err = c.UDPLikeConn.Close()
	})
	return
}
