package measurex

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// Logger is the logger type we use.
type Logger interface {
	netxlite.Logger

	Info(msg string)
	Infof(format string, v ...interface{})
}

// HoldingLogger is a Logger that holds messages for a bunch
// of milliseconds and thene emits them in a batch.
//
// This kind of logger improves the UX in case there are many
// timeouts and doesn't overwhelm the screen otherwise.
//
// Make sure you call HoldingLogger.Stop when done with it.
type HoldingLogger struct {
	begin  time.Time
	cancel context.CancelFunc
	ch     chan *holdingLoggerEntry
	fin    chan interface{}
	logger Logger
	once   *sync.Once
}

type holdingLoggerEntry struct {
	f   func(msg string)
	msg string
	t   time.Duration
}

// Debug implements Logger.Debug.
func (hl *HoldingLogger) Debug(message string) {
	hl.ch <- &holdingLoggerEntry{
		f:   hl.logger.Debug,
		msg: message,
		t:   time.Since(hl.begin),
	}
}

// Debugf implements Logger.Debugf.
func (hl *HoldingLogger) Debugf(format string, v ...interface{}) {
	hl.Debug(fmt.Sprintf(format, v...))
}

// Info implements Logger.Info.
func (hl *HoldingLogger) Info(message string) {
	hl.ch <- &holdingLoggerEntry{
		f:   hl.logger.Info,
		msg: message,
		t:   time.Since(hl.begin),
	}
}

// Infof implements Logger.Infof.
func (hl *HoldingLogger) Infof(format string, v ...interface{}) {
	hl.Info(fmt.Sprintf(format, v...))
}

// NewHoldingLogger is a factory that creates a new HoldingLogger
// using the given logger for emitting messages.
func NewHoldingLogger(logger Logger) *HoldingLogger {
	ctx, cancel := context.WithCancel(context.Background())
	hl := &HoldingLogger{
		begin:  time.Now(),
		cancel: cancel,
		ch:     make(chan *holdingLoggerEntry),
		fin:    make(chan interface{}),
		logger: logger,
		once:   &sync.Once{},
	}
	go hl.loop(ctx)
	return hl
}

// Stop stops the HoldingLogger and waits for the
// background goroutine to terminate logging.
func (hl *HoldingLogger) Stop() {
	hl.once.Do(func() {
		hl.cancel()
		<-hl.fin
	})
}

// loop runs the HoldingLogger main loop
func (hl *HoldingLogger) loop(ctx context.Context) {
	interval := 500 * time.Millisecond
	var all []*holdingLoggerEntry
	defer close(hl.fin)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			hl.emit(all)
			return
		case <-ticker.C:
			hl.emit(all)
		case entry := <-hl.ch:
			all = append(all, entry)
		}
	}
}

// emit emits all the messages inside `all`
func (hl *HoldingLogger) emit(all []*holdingLoggerEntry) {
	for _, entry := range all {
		entry.f(fmt.Sprintf("[%8.3f] %s", entry.t.Seconds(), entry.msg))
	}
}
