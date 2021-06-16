// Package tlshandshaker contains a TLS handshaker service. This service will
// use the system TLS handshaker by default.
//
// You should create a service instance using New. Then, you should
// start a bunch of background goroutines implementing the service
// using the StartN method. At this point, you may issue as many handshakes
// as you want using the Handshake method. When you are done, you
// can finally call the Stop method to stop the service.
package tlshandshaker

import (
	"context"
	"crypto/tls"
	"net"
	"sync"

	"github.com/ooni/probe-cli/v3/internal/engine/netx/tlsdialer"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/trace"
)

// Service is the TLS handshaker service. Use new to create a
// correctly-initialized new instance of Service.
type Service struct {
	// input is the input channel of the background goroutine.
	input chan *HandshakeRequest

	// once allows for once semantics in Stop.
	once sync.Once

	// wg is the wait group to ensure we join all goroutines.
	wg sync.WaitGroup
}

// New creates a new instance of Service.
func New() *Service {
	return &Service{
		input: make(chan *HandshakeRequest),
		once:  sync.Once{},
		wg:    sync.WaitGroup{},
	}
}

// StartN starts N instances of the background servicing goroutine. If
// count is zero or negative, we won't start any instance. This function
// may be called at any time to add more background goroutines.
func (svc *Service) StartN(count int) {
	for i := 0; i < count; i++ {
		svc.wg.Add(1)
		go svc.mainloop()
	}
}

// Stop stops all the background goroutines. This function is
// idempotent and does not cause any data race.
func (svc *Service) Stop() {
	svc.once.Do(func() {
		close(svc.input)
	})
}

// Logger is the logger expected by this package.
type Logger interface {
	// Debugf formats and emits a debug message.
	Debugf(format string, v ...interface{})

	// Debug emits a debug message.
	Debug(message string)

	// Infof formats and emits an informational message.
	Infof(format string, v ...interface{})

	// Info emits an informational message.
	Info(message string)

	// Warnf formats and emits a warning message.
	Warnf(format string, v ...interface{})

	// Warn emits a warning message.
	Warn(message string)
}

// HandshakeRequest contains settings for Service.Handshake. You MUST fill
// in any field marked as MANDATORY before calling Service.Handshake.
type HandshakeRequest struct {
	// Conn is the MANDATORY TCP connection to use. This service WILL NOT
	// take ownership of this connection. You SHOULD close this connection
	// if the handshake fails, unless there is a reason to keep it open.
	Conn net.Conn

	// Config is the MANDATORY TLS config.
	Config *tls.Config

	// Logger is the optional logger to use. If this field is nil,
	// then there there won't be any logging for this request.
	Logger Logger

	// Saver is the optional saver to use. If this field is nil,
	// then we won't save any event.
	Saver *trace.Saver

	// ctx is the context to use.
	ctx context.Context

	// output is the output channel of the background goroutine.
	output chan *handshakeResponse
}

// handshakeResponse is the response after calling Service.Handshake.
type handshakeResponse struct {
	// Conn is a TLS connection or a nil pointer.
	Conn net.Conn

	// Err is the resulting error or nil.
	Err error
}

// Handshake performs a TLS handshake in the background. You MUST NOT
// call this method before StartN or after Stop.
func (svc *Service) Handshake(
	ctx context.Context, req *HandshakeRequest) (net.Conn, error) {
	req.ctx = ctx
	req.output = make(chan *handshakeResponse)
	svc.input <- req
	resp := <-req.output
	return resp.Conn, resp.Err
}

// mainloop runs the main loop of the service.
func (svc *Service) mainloop() {
	defer svc.wg.Done()
	for req := range svc.input {
		req.output <- svc.handshake(req)
	}
}

// handshake implements the handshake operation.
func (svc *Service) handshake(req *HandshakeRequest) *handshakeResponse {
	th := svc.newTLSHandshaker(req)
	resp := &handshakeResponse{}
	conn, _, err := th.Handshake(req.ctx, req.Conn, req.Config)
	if err != nil {
		resp.Err = err
		return resp
	}
	resp.Conn = conn
	return resp
}

// newTLSHandshaker creates a new TLS handshaker based on the content of req.
func (svc *Service) newTLSHandshaker(req *HandshakeRequest) tlsdialer.TLSHandshaker {
	// TODO(bassosimone): this code should live inside netx.
	var th tlsdialer.TLSHandshaker
	th = &tlsdialer.SystemTLSHandshaker{}
	th = &tlsdialer.ErrorWrapperTLSHandshaker{TLSHandshaker: th}
	if req.Logger != nil {
		th = &tlsdialer.LoggingTLSHandshaker{
			TLSHandshaker: th,
			Logger:        req.Logger,
		}
	}
	if req.Saver != nil {
		th = &tlsdialer.SaverTLSHandshaker{
			TLSHandshaker: th,
			Saver:         req.Saver,
		}
	}
	return th
}
