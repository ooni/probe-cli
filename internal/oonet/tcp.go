package oonet

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"
)

// DefaultDialer is the dialer used by Transport.DefaultDialContext.
var DefaultDialer = &net.Dialer{
	// Timeout is the connect timeout.
	Timeout: 15 * time.Second,

	// KeepAlive is the keep-alive interval (may not work on all
	// platforms, as it depends on kernel support).
	KeepAlive: 30 * time.Second,
}

// ErrDial is an error occurred when dialing.
type ErrDial struct {
	error
}

// Unwrap returns the underlying error.
func (err *ErrDial) Unwrap() error {
	return err.error
}

// ConnWrapper is a wrapper for net.ConnWrapper.
type ConnWrapper struct {
	net.Conn
}

// ErrRead is a read error.
type ErrRead struct {
	error
}

// Unwrap returns the underlying error.
func (err *ErrRead) Unwrap() error {
	return err.error
}

// Read implements net.Conn.Read. When this function returns an
// error it's always an ErrRead error.
func (conn *ConnWrapper) Read(b []byte) (int, error) {
	count, err := conn.Conn.Read(b)
	if err != nil {
		return 0, &ErrRead{err}
	}
	return count, nil
}

// ErrWrite is a write error.
type ErrWrite struct {
	error
}

// Unwrap returns the underlying error.
func (err *ErrWrite) Unwrap() error {
	return err.error
}

// Write implements net.Conn.Write. When this function returns an
// error, it's always an ErrWrite error.
func (conn *ConnWrapper) Write(b []byte) (int, error) {
	count, err := conn.Conn.Write(b)
	if err != nil {
		return 0, &ErrWrite{err}
	}
	return count, nil
}

// ErrProxyNotImplemented indicates that the dialing function does not support
// the specific Proxy configured using the Transport.Proxy field.
var ErrProxyNotImplemented = errors.New("oonet: proxy not implemented")

// DialContext dials a cleartext connection. This function returns an ErrDial
// instance in case of failure. This function will use the Transport.Logger
// logger to emit logs. This function will either use the DefaultDialContext,
// by default, or ContextOverrides().DialContext, if configured. When using
// this function, the underlying error will be ErrProxyNotImplemented, if the
// Transport.Proxy field is not nil. We currently do not support dialing a
// generic network connection using any kind of proxy. (We do support proxies
// when internally dialing connections for HTTP's sake though.) On success,
// the returned net.Conn will be wrapped using ConnWrapper.
func (txp *Transport) DialContext(
	ctx context.Context, network string, addr string) (net.Conn, error) {
	if txp.Proxy != nil {
		return nil, &ErrDial{ErrProxyNotImplemented}
	}
	return txp.httpDialContext(ctx, network, addr)
}

// httpDialContext is the DialContext called by HTTP code to bypass the check on the
// presence of the Transport.Proxy field implemented by Transport.DialContext.
func (txp *Transport) httpDialContext(
	ctx context.Context, network string, addr string) (net.Conn, error) {
	log := txp.logger()
	log.Debugf("dial: %s/%s...", addr, network)
	conn, err := txp.routeDialContext(ctx, network, addr)
	if err != nil {
		log.Debugf("dial: %s/%s... %s", addr, network, err)
		return nil, &ErrDial{err}
	}
	log.Debugf("dial: %s/%s... ok", addr, network)
	return &ConnWrapper{conn}, nil
}

// routeDialContext routes the DialContext call.
func (txp *Transport) routeDialContext(
	ctx context.Context, network string, addr string) (net.Conn, error) {
	if overrides := ContextOverrides(ctx); overrides != nil && overrides.DialContext != nil {
		return overrides.DialContext(ctx, network, addr)
	}
	return txp.DefaultDialContext(ctx, network, addr)
}

// ErrConnect is an error occurred when TCP-connecting.
type ErrConnect struct {
	// Errors contains all the errors that occurred.
	Errors []error
}

// Error implements error.Error.
func (err *ErrConnect) Error() string {
	return fmt.Sprintf("one or more connect() failed: %#v", err.Errors)
}

// DefaultDialContext is the default DialContext implementation. When there is an error
// when connecting, this function will return an ErrConnect instance.
func (txp *Transport) DefaultDialContext(
	ctx context.Context, network string, addr string) (net.Conn, error) {
	hostname, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	ipaddrs, err := txp.LookupHost(ctx, hostname)
	if err != nil {
		return nil, err
	}
	aggregate := &ErrConnect{}
	for _, ipaddr := range ipaddrs {
		epnt := net.JoinHostPort(ipaddr, port)
		conn, err := DefaultDialer.DialContext(ctx, network, epnt)
		if err == nil {
			return conn, nil
		}
		aggregate.Errors = append(aggregate.Errors, err)
	}
	return nil, aggregate
}
