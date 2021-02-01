package hhfm_test

import (
	"context"
	"io/ioutil"
	"net"
	"net/http"
	"time"
)

type FakeDialer struct {
	Conn net.Conn
	Err  error
}

func (d FakeDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	time.Sleep(10 * time.Microsecond)
	return d.Conn, d.Err
}

type FakeTransport struct {
	Err  error
	Func func(*http.Request) (*http.Response, error)
	Resp *http.Response
}

func (txp FakeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	time.Sleep(10 * time.Microsecond)
	if txp.Func != nil {
		return txp.Func(req)
	}
	if req.Body != nil {
		ioutil.ReadAll(req.Body)
		req.Body.Close()
	}
	if txp.Err != nil {
		return nil, txp.Err
	}
	txp.Resp.Request = req // non thread safe but it doesn't matter
	return txp.Resp, nil
}

func (txp FakeTransport) CloseIdleConnections() {}

type FakeBody struct {
	Err error
}

func (fb FakeBody) Read(p []byte) (int, error) {
	time.Sleep(10 * time.Microsecond)
	return 0, fb.Err
}

func (fb FakeBody) Close() error {
	return nil
}
