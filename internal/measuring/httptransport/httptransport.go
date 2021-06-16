// Package httptransport contains the HTTP round tripper service. This
// service will use a single connection per request. There are no persistent
// connections here. Also, there are no redirects. You need to implement
// the redirection yourself, if you need this functionality.
//
// You should create a service instance using New. Then, you should
// start a bunch of background goroutines implementing the service
// using the StartN method. At this point, you may issue as many queries
// as you want using the LookupHost method. When you are done, you
// can finally call the Stop method to stop the service.
package httptransport

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"sync"

	"github.com/ooni/probe-cli/v3/internal/engine/netx/httptransport"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/trace"
)

// Service is the HTTP round tripper service. Use new to create a
// correctly-initialized new instance of Service.
type Service struct {
	// input is the input channel of the background goroutine.
	input chan *RoundTripRequest

	// once allows for once semantics in Stop.
	once sync.Once

	// wg is the wait group to ensure we join all goroutines.
	wg sync.WaitGroup
}

// New creates a new instance of Service.
func New() *Service {
	return &Service{
		input: make(chan *RoundTripRequest),
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

// RoundTripRequest contains settings for Service.RoundTrip. You MUST fill
// in any field marked as MANDATORY before calling Service.RoundTrip.
type RoundTripRequest struct {
	// Req is the MANDATORY request to send.
	Req *http.Request

	// Conn is the MANDATORY connection to use.
	Conn net.Conn

	// Logger is the optional logger to use. If this field is nil,
	// then there there won't be any logging for this request.
	Logger Logger

	// Saver is the optional saver to use. If this field is nil,
	// then we won't save any event.
	Saver *trace.Saver

	// ctx is the context to use.
	ctx context.Context

	// output is the output channel of the background goroutine.
	output chan *roundTripResponse
}

// roundTripResponse is the response after calling Service.RoundTrip.
type roundTripResponse struct {
	// Resp is either the response or nil.
	Resp *http.Response

	// Err is the resulting error or nil.
	Err error
}

// RoundTrip performs an HTTP roundTrip in the background. You MUST NOT
// call this method before StartN or after Stop.
func (svc *Service) RoundTrip(
	ctx context.Context, req *RoundTripRequest) (*http.Response, error) {
	req.ctx = ctx
	req.output = make(chan *roundTripResponse)
	svc.input <- req
	resp := <-req.output
	return resp.Resp, resp.Err
}

// mainloop runs the main loop of the service.
func (svc *Service) mainloop() {
	defer svc.wg.Done()
	for req := range svc.input {
		req.output <- svc.roundTrip(req)
	}
}

// roundTrip implements the HTTP roundTrip operation.
func (svc *Service) roundTrip(req *RoundTripRequest) *roundTripResponse {
	txp := svc.newRoundTripper(req)
	defer txp.CloseIdleConnections()
	resp := &roundTripResponse{}
	httpResp, err := txp.RoundTrip(req.Req.WithContext(req.ctx))
	if err != nil {
		resp.Err = err
		return resp
	}
	resp.Resp = httpResp
	return resp
}

// newRoundTripper creates a new HTTP round tripper based on req.
func (svc *Service) newRoundTripper(req *RoundTripRequest) httptransport.RoundTripper {
	var txp httptransport.RoundTripper = &http.Transport{
		Proxy: nil,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return req.Conn, nil
		},
		DialTLSContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return req.Conn, nil
		},
		TLSClientConfig:     &tls.Config{},
		TLSHandshakeTimeout: 0,
		DisableKeepAlives:   true,
		DisableCompression:  true,
		ForceAttemptHTTP2:   true,
	}
	if req.Logger != nil {
		txp = &httptransport.LoggingTransport{
			RoundTripper: txp,
			Logger:       req.Logger,
		}
	}
	if req.Saver != nil {
		txp = &httptransport.SaverMetadataHTTPTransport{
			RoundTripper: txp,
			Saver:        req.Saver,
			Transport:    "tcp",
		}
	}
	return txp
}
