// Package resolver contains a resolver service. This resolver will
// use the system resolver (i.e., getaddrinfo on Unix).
//
// You should create a service instance using New. Then, you should
// start a bunch of background goroutines implementing the service
// using the StartN method. At this point, you may issue as many queries
// as you want using the LookupHost method. When you are done, you
// can finally call the Stop method to stop the service.
package resolver

import (
	"context"
	"sync"

	"github.com/ooni/probe-cli/v3/internal/engine/netx/resolver"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/trace"
)

// Service is the resolver service. Use new to create a
// correctly-initialized new instance of Service.
type Service struct {
	// input is the input channel of the background goroutine.
	input chan *LookupHostRequest

	// once allows for once semantics in Stop.
	once sync.Once

	// wg is the wait group to ensure we join all goroutines.
	wg sync.WaitGroup
}

// New creates a new instance of Service.
func New() *Service {
	return &Service{
		input: make(chan *LookupHostRequest),
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

// LookupHostRequest contains settings for Service.LookupHost. You MUST fill
// in any field marked as MANDATORY before calling Service.LookupHost.
type LookupHostRequest struct {
	// Domain is the MANDATORY address to resolve.
	Domain string

	// Logger is the optional logger to use. If this field is nil,
	// then there there won't be any logging for this request.
	Logger Logger

	// Saver is the optional saver to use. If this field is nil,
	// then we won't save any event.
	Saver *trace.Saver

	// ctx is the context to use.
	ctx context.Context

	// output is the output channel of the background goroutine.
	output chan *lookupHostResponse
}

// lookupHostResponse is the response after calling Service.LookupHost.
type lookupHostResponse struct {
	// Addresses contains the resolved addresses or is nil.
	Addresses []string

	// Err is the resulting error or nil.
	Err error
}

// LookupHost performs a lookupHost in the background. You MUST NOT
// call this method before StartN or after Stop.
func (svc *Service) LookupHost(
	ctx context.Context, req *LookupHostRequest) ([]string, error) {
	req.ctx = ctx
	req.output = make(chan *lookupHostResponse)
	svc.input <- req
	resp := <-req.output
	return resp.Addresses, resp.Err
}

// mainloop runs the main loop of the service.
func (svc *Service) mainloop() {
	defer svc.wg.Done()
	for req := range svc.input {
		req.output <- svc.lookupHost(req)
	}
}

// lookupHost implements the lookupHost operation.
func (svc *Service) lookupHost(req *LookupHostRequest) *lookupHostResponse {
	r := svc.newResolver(req)
	resp := &lookupHostResponse{}
	addrs, err := r.LookupHost(req.ctx, req.Domain)
	if err != nil {
		resp.Err = err
		return resp
	}
	resp.Addresses = addrs
	return resp
}

// newResolver creates a new resolver based on the content of req.
func (svc *Service) newResolver(req *LookupHostRequest) resolver.Resolver {
	// TODO(bassosimone): this code should live inside netx.
	var r resolver.Resolver
	r = &resolver.SystemResolver{}
	r = &resolver.ErrorWrapperResolver{Resolver: r}
	if req.Logger != nil {
		r = &resolver.LoggingResolver{
			Resolver: r,
			Logger:   req.Logger,
		}
	}
	if req.Saver != nil {
		r = &resolver.SaverResolver{
			Resolver: r,
			Saver:    req.Saver,
		}
	}
	return r
}
