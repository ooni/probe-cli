package session

//
// Public definition of Session
//

import (
	"context"
	"errors"
	"log"
	"sync"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// Session is a measurement session. The zero value of this structure
// is invalid. You must use the [New] factory to create a new valid instance.
//
// A session consists of a background goroutine to which you Send
// [Request]s. Each [Request] causes the background goroutine to
// start running a long-running task. You should Recv [Event]s
// emitted by the task until you find the matching [Event] that implies
// that the task you started has finished running.
//
// While running, a task emits "ticker" events that you can use to
// fill a progress bar and to decide when the task should stop running. To
// stop a long-running task, you call [Close], which forces the background
// goroutine to stop as soon as possible.
//
// Once a [Session] has been terminated using [Close] you loose all
// the [Session] state and you must create a new [Session].
type Session struct {
	// cancel allows us to terminate the backround goroutine.
	cancel context.CancelFunc

	// input is the channel to send [Request] to the background goroutine.
	input chan *Request

	// once allows us to cleanup the state just once.
	once sync.Once

	// output is the channel from which we read the emitted [Event]s.
	output chan *Event

	// state is the background goroutine's state, which only
	// the background goroutine is allowed to modify. We start
	// with a nil state and create state using the bootstrap
	// long running task.
	state model.OptionalPtr[state]

	// terminated is closed when the background goroutine terminates.
	terminated chan any
}

// New creates a new measurement [Session]. This function will create
// a background goroutine that will handle incoming [Request]s.
func New() *Session {
	ctx, cancel := context.WithCancel(context.Background())
	// Implementation note: we use buffered channels for input and
	// output to avoid loosing events in _most_ cases.
	s := &Session{
		cancel:     cancel,
		input:      make(chan *Request, 1024),
		once:       sync.Once{},
		output:     make(chan *Event, 1024),
		state:      model.OptionalPtr[state]{},
		terminated: make(chan any),
	}
	go s.mainloop(ctx)
	return s
}

// Request requests a [Session] to perform a background task. This
// struct contains several pointers. Each of them indicates a specific
// task the [Session] should run. The [Session] will go through each
// pointer and only consider the first one that is not nil. As such
// it's pointless to initialize more than one pointer.
type Request struct {
	// Bootstrap indicates that the [Session] should bootstrap
	// and contains bootstrap configuration.
	Bootstrap *BootstrapRequest

	// CheckIn indicates that the [Session] should call the check-in API
	// and only works if you have already bootstrapped.
	CheckIn *CheckInRequest

	// Geolocate indicates that the [Session] should obtain
	// the current probe's geolocation. This task requires you
	// to successfully bootstrap a session first.
	Geolocate *GeolocateRequest

	// Submit indicates that the [Session] should submit a
	// measurement. You must bootstrap first.
	Submit *SubmitRequest

	// WebConnectivity indicates that the [Session] should
	// run the Web Connectivity experiment. You must bootstrap first.
	WebConnectivity *WebConnectivityRequest
}

// ErrSessionTerminated indicates that the background goroutine has terminated.
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

// Close stops the goroutine running in the background and releases
// all the resources allocated by the [Session].
func (s *Session) Close() error {
	s.once.Do(s.joincleanup)
	return nil
}

// joincleanup joins the [Session] and closes open resources.
func (s *Session) joincleanup() {
	s.cancel()
	<-s.terminated
	if s.state.IsSome() {
		s.state.Unwrap().cleanup()
		s.state = model.OptionalPtr[state]{} // just to be tidy
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

// maybeEmit emits an [Event] if possible. If the output channel
// buffer is full, we are not going to emit the event. In such
// a case, we will print a log message to the standard error file.
func (s *Session) maybeEmit(ev *Event) {
	select {
	case s.output <- ev:
	default:
		log.Printf("session: cannot send event: %+v", ev)
	}
}

// Event is an event emitted by a [Session]. Only one of the pointers
// of the [Event] will be set. The pointer being set uniquely identifies
// the specific event that has occurred.
type Event struct {
	// Bootstrap is emitted at the end of the bootstrap.
	Bootstrap *BootstrapEvent

	// CheckIn is the event emitted at the end of the check-in.
	CheckIn *CheckInEvent

	// Geolocate is emitted at the end of geolocate.
	Geolocate *GeolocateEvent

	// Log is a log event. Any task will emit log events.
	Log *LogEvent

	// Progress is a progress event. Only experiments that print
	// their own progress will emit this event.
	Progress *ProgressEvent

	// Submit is emitted after a measurement submission.
	Submit *SubmitEvent

	// Ticker is a ticker event. Any task will emit ticker
	// events so that you can increase a progress bar and
	// decide whether the task should be stopped.
	Ticker *TickerEvent

	// WebConnectivity is the Web Connectivity event, emitted
	// once we finished measuring a given URL.
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
	if req.Bootstrap != nil {
		s.bootstrap(ctx, req.Bootstrap)
		return
	}
	if req.CheckIn != nil {
		s.checkin(ctx, req.CheckIn)
		return
	}
	if req.Geolocate != nil {
		s.geolocate(ctx, req.Geolocate)
		return
	}
	if req.Submit != nil {
		s.submit(ctx, req.Submit)
		return
	}
	if req.WebConnectivity != nil {
		s.webconnectivity(ctx, req.WebConnectivity)
		return
	}
}
