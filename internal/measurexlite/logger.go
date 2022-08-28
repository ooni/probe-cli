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
// about an in-progress operation. If it takes too much
// time to emit the result of the operation, the code
// will emit an interim log message mentioning that the
// operation is currently in progress.
func NewOperationLogger(logger model.Logger, format string, v ...any) *OperationLogger {
	return newOperationLogger(500*time.Millisecond, logger, format, v...)
}

func newOperationLogger(maxwait time.Duration, logger model.Logger, format string, v ...any) *OperationLogger {
	ol := &OperationLogger{
		logger:  logger,
		maxwait: maxwait,
		message: fmt.Sprintf(format, v...),
		once:    &sync.Once{},
		sighup:  make(chan any),
		wg:      &sync.WaitGroup{},
	}
	ol.wg.Add(1)
	go ol.maybeEmitProgress()
	return ol
}

// OperationLogger keeps state required to log about an in-progress
// operation as documented by [NewOperationLogger].
type OperationLogger struct {
	logger  model.Logger
	maxwait time.Duration
	message string
	once    *sync.Once
	sighup  chan any
	wg      *sync.WaitGroup
}

func (ol *OperationLogger) maybeEmitProgress() {
	defer ol.wg.Done()
	timer := time.NewTimer(ol.maxwait)
	defer timer.Stop()
	select {
	case <-timer.C:
		ol.logger.Infof("%s... in progress", ol.message)
	case <-ol.sighup:
		// we'll emit directly in stop
	}
}

// Stop must be called when the operation is done. The [err] argument
// is the result of the operation, which may be nil. This method ensures
// that we log the final result of the now-completed operation.
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
