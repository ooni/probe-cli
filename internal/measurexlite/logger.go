package measurexlite

//
// Logging support
//

import (
	"fmt"
	"sync"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// NewOperationLogger creates a new logger that logs
// about an in-progress operation.
func NewOperationLogger(logger model.Logger, format string, v ...interface{}) *OperationLogger {
	ol := &OperationLogger{
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

// OperationLogger logs about an in-progress operation
type OperationLogger struct {
	logger  model.Logger
	message string
	once    *sync.Once
	sighup  chan interface{}
	wg      *sync.WaitGroup
}

func (ol *OperationLogger) logloop() {
	defer ol.wg.Done()
	timer := time.NewTimer(500 * time.Millisecond)
	defer timer.Stop()
	select {
	case <-timer.C:
		ol.logger.Infof("%s... in progress", ol.message)
	case <-ol.sighup:
		// we'll emit directly in stop
	}
}

func (ol *OperationLogger) Stop(err error) {
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
