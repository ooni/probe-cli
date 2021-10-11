package ooshell

//
// callbacks.go
//
// Code to deal with experiment callbacks
//

// callbacksNull will do nothing in each callback.
type callbacksNull struct{}

func (cb *callbacksNull) OnProgress(percentage float64, message string) {
	// nothing
}

// callbacksReportBack will report back to the caller.
type callbacksReportBack struct {
	// exp is the experiment that owns us
	exp *experimentDB
}

func (cb *callbacksReportBack) OnProgress(percentage float64, message string) {
	cb.exp.handleProgress(percentage, message)
}
