// Package connector contains a connector service. This connector will
// assume you pass to it IP addresses to connect to. If you pass domain
// names, it will work, but this is not the main intended usage. (In
// such a case, it will use a &net.Resolver{} instance to perform domain
// name resolutions.)
//
// You should create a service instance using New. Then, you should
// start a bunch of background goroutines implementing the service
// using the StartN method. At this point, you may issue as many connects
// as you want using the DialContext method. When you are done, you
// can finally call the Stop method to stop the service.
package connector

import (
	"context"
	"net"
	"sync"

	"github.com/ooni/probe-cli/v3/internal/engine/netx/dialer"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/trace"
)

// Service is the connector service. Use new to create a
// correctly-initialized new instance of Service.
type Service struct {
	// input is the input channel of the background goroutine.
	input chan *DialRequest

	// once allows for once semantics in Stop.
	once sync.Once

	// wg is the wait group to ensure we join all goroutines.
	wg sync.WaitGroup
}

// New creates a new instance of Service.
func New() *Service {
	return &Service{
		input: make(chan *DialRequest),
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

// DialRequest contains settings for Service.DialContext. You MUST fill in
// any field marked as MANDATORY before calling Service.DialContext.
type DialRequest struct {
	// Network is the MANDATORY network to use.
	Network string

	// Address is the MANDATORY address to connect to.
	Address string

	// Logger is the optional logger to use. If this field is nil,
	// then there there won't be any logging for this request.
	Logger Logger

	// Saver is the optional saver to use. If this field is nil,
	// then we won't save any event.
	Saver *trace.Saver

	// ctx is the context to use.
	ctx context.Context

	// output is the output channel of the background goroutine.
	output chan *dialResponse
}

// dialResponse is the response after calling Service.DialContext.
type dialResponse struct {
	// Conn is an established connection or nil.
	Conn net.Conn

	// Err is the resulting error or nil.
	Err error
}

// DialContext performs a dialContext in the background. You MUST NOT
// call this method before StartN or after Stop.
func (svc *Service) DialContext(
	ctx context.Context, req *DialRequest) (net.Conn, error) {
	req.ctx = ctx
	req.output = make(chan *dialResponse)
	svc.input <- req
	resp := <-req.output
	return resp.Conn, resp.Err
}

// mainloop runs the main loop of the service.
func (svc *Service) mainloop() {
	defer svc.wg.Done()
	for req := range svc.input {
		req.output <- svc.dialContext(req)
	}
}

// dialContext implements the dialContext operation.
func (svc *Service) dialContext(req *DialRequest) *dialResponse {
	dr := &net.Resolver{} // as documented
	d := dialer.New(&dialer.Config{
		ContextByteCounting: true,
		DialSaver:           req.Saver,
		Logger:              req.Logger,
		ReadWriteSaver:      req.Saver,
	}, dr)
	resp := &dialResponse{}
	conn, err := d.DialContext(req.ctx, req.Network, req.Address)
	if err != nil {
		resp.Err = err
		return resp
	}
	resp.Conn = conn
	return resp
}
