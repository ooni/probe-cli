// Package session implements a measurement session. The design of
// this package is such that we can split the measurement engine proper
// and the application using it. In particular, this design is such
// that it would be easy to expose this API as a C library.
//
// The general usage of this package is the following:
//
// 1. XXX document
//
// Go packages should not use this package directly but rather use
// the sessionclient package, which provides idiomatic wrappers.
package session

import (
	"context"
	"errors"
	"log"
	"sync"
)

// Session is a measurement session.
type Session struct {
	// cancel allows us to terminate the backround goroutine.
	cancel context.CancelFunc

	// input is the channel to send input to the background goroutine.
	input chan *Request

	// once allows us to run cleanups just once.
	once sync.Once

	// output is the channel from which we read the emitted events.
	output chan *Event

	// state is the background goroutine's state, which only
	// the background goroutine is allowed to modify.
	state *state

	// terminated is closed when the background goroutine terminates.
	terminated chan any
}

// New creates a new measurement [Session]. This function will create
// a background goroutine that will handle incoming [Request]s.
func New() *Session {
	ctx, cancel := context.WithCancel(context.Background())
	s := &Session{
		cancel:     cancel,
		input:      make(chan *Request, 1024),
		once:       sync.Once{},
		output:     make(chan *Event, 1024),
		state:      nil,
		terminated: make(chan any),
	}
	go s.mainloop(ctx)
	return s
}

// Request requests a [Session] to perform a background task. This
// struct contains several pointers. Each of them indicates a specific
// task the [Session] should run. The [Session] will go through each
// pointer and only consider the first one that is not nil.
type Request struct {
	// Bootstrap indicates that the [Session] should bootstrap
	// and contains bootstrap configuration.
	Bootstrap *BootstrapRequest

	// CheckIn indicates that the [Session] should call the check-in API.
	CheckIn *CheckInRequest

	// Geolocate indicates that the [Session] should obtain
	// the current probe's geolocation.
	Geolocate *GeolocateRequest

	// Submit indicates that the [Session] should submit a measurement.
	Submit *SubmitRequest

	// WebConnectivity indicates that the [Session] should
	// run the Web Connectivity experiment.
	WebConnectivity *WebConnectivityRequest
}

// ErrSessionTerminated indicates that the background goroutine
// servicing the [Session] [Request]s has stopped.
var ErrSessionTerminated = errors.New("session: terminated")

// Send sends a [Request] to a [Session]. This function will return an
// error when the [Session] has been stopped or when the context has expired.
func (s *Session) Send(ctx context.Context, req *Request) error {
	select {
	case s.input <- req:
		return nil
	case <-s.terminated:
		return ErrSessionTerminated
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Close stops the goroutine running in the background and release
// all the resources allocated by the [Session].
func (s *Session) Close() error {
	s.once.Do(s.joincleanup)
	return nil
}

// joincleanup joins the [Session] and closes open resources.
func (s *Session) joincleanup() {
	s.cancel()
	<-s.terminated
	if s.state != nil {
		s.state.cleanup()
	}
}

// Recv receives the next [Event] emitted by a [Session]. This function
// will return an error when the [Session] has been stopped or when
// the context has expired.
func (s *Session) Recv(ctx context.Context) (*Event, error) {
	select {
	case ev := <-s.output:
		return ev, nil
	case <-s.terminated:
		return nil, ErrSessionTerminated
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// emit emits an [Event].
func (s *Session) emit(ev *Event) {
	select {
	case s.output <- ev:
	default:
		log.Printf("session: cannot send event: %+v", ev)
	}
}

// Event is an event emitted by a [Session].
type Event struct {
	// Bootstrap is emitted at the end of the bootstrap.
	Bootstrap *BootstrapEvent

	// CheckIn is the event emitted at the end of the check-in.
	CheckIn *CheckInEvent

	// Geolocate is emitted at the end of geolocate.
	Geolocate *GeolocateEvent

	// Log is a log event.
	Log *LogEvent

	// Progress is a progress event.
	Progress *ProgressEvent

	// Submit is emitted after a measurement submission.
	Submit *SubmitEvent

	// Ticker is a ticker event.
	Ticker *TickerEvent

	// WebConnectivity is the Web Connectivity event.
	WebConnectivity *WebConnectivityEvent
}

// mainloop runs the [Session] main loop.
func (s *Session) mainloop(ctx context.Context) {
	defer close(s.terminated)
	for {
		select {
		case <-ctx.Done():
			return
		case req := <-s.input:
			s.handle(ctx, req)
		}
	}
}

// handle handles an incoming [Request].
func (s *Session) handle(ctx context.Context, req *Request) {
	// TODO(bassosimone): rewrite trying to avoid all these ifs.
	if req.Bootstrap != nil {
		s.bootstrap(ctx, req)
		return
	}
	if req.CheckIn != nil {
		s.checkin(ctx, req)
		return
	}
	if req.Geolocate != nil {
		s.geolocate(ctx, req)
		return
	}
	if req.Submit != nil {
		s.submit(ctx, req)
		return
	}
	if req.WebConnectivity != nil {
		s.webconnectivity(ctx, req)
		return
	}
}
