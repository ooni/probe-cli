package main

//
// test is a mock task to mimic the request-response API for the FFI consumer.
//

import (
	"context"
	"errors"
)

func init() {
	taskRegistry["Test"] = &testTaskRunner{}
}

var (
	errTestDisabled = errors.New("request argument for test disabled")
)

// testOptions contains the request options for the Test task.
type testOptions struct {
	Test bool `json:",omitempty"`
}

// testResponse is the response for the Test task.
type testResponse struct {
	Response string `json:",omitempty"`
	Error    string `json:"omitempty"`
}

type testTaskRunner struct{}

var _ taskRunner = &testTaskRunner{}

// main implements taskRunner.main
func (tr *testTaskRunner) main(ctx context.Context, emitter taskMaybeEmitter,
	req *request, res *response) {
	logger := newTaskLogger(emitter, false)
	if !req.Test.Test {
		logger.Warnf("task_runner: %s", errTestDisabled.Error())
		res.Test.Error = errTestDisabled.Error()
		return
	}
	logger.Info("task_runner: a log event for the Test task")
	res.Test.Response = "test success"
}
