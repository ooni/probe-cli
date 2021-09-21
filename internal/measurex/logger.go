package measurex

import (
	"fmt"
	"sync"
	"time"

	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// Logger is the logger type we use. This type is compatible
// with the logger type of github.com/apex/log.
type Logger interface {
	netxlite.Logger

	Info(msg string)
	Infof(format string, v ...interface{})
}

// newOperationLogger creates a new logger that logs
// about an in-progress operation.
func newOperationLogger(logger Logger, format string, v ...interface{}) *operationLogger {
	ol := &operationLogger{
		sighup:  make(chan interface{}),
		logger:  logger,
		once:    &sync.Once{},
		message: fmt.Sprintf(format, v...),
		wg:      &sync.WaitGroup{},
	}
	ol.wg.Add(1)
	go ol.logloop()
	return ol
}

// operationLogger logs about an in-progress operation
type operationLogger struct {
	logger  Logger
	message string
	once    *sync.Once
	sighup  chan interface{}
	wg      *sync.WaitGroup
}

func (ol *operationLogger) logloop() {
	defer ol.wg.Done()
	timer := time.NewTimer(500 * time.Millisecond)
	defer timer.Stop()
	select {
	case <-timer.C:
	case <-ol.sighup:
		return
	}
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		ol.logger.Infof("%s... in progress", ol.message)
		select {
		case <-ol.sighup:
			return
		case <-ticker.C:
			// continue the loop
		}
	}
}

func (ol *operationLogger) Stop(err error) {
	ol.once.Do(func() {
		close(ol.sighup)
		ol.wg.Wait()
		if err != nil {
			ol.logger.Infof("%s... %s", ol.message, err.Error())
			return
		}
		ol.logger.Infof("%s... ok", ol.message)
	})
}
