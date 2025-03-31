package motor

//
// test is a mock task to mimic the request-response API for the FFI consumer.
//

import (
	"context"
	"encoding/json"
	"errors"
)

func init() {
	taskRegistry["Test"] = &testTaskRunner{}
}

var (
	errTestDisabled = errors.New("request argument for test disabled")

	errParseFailed = errors.New("unable to parse task arguments")
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
	req *Request, resp *Response) {
	logger := newTaskLogger(emitter, false)
	args := &testOptions{}
	if err := json.Unmarshal(req.Arguments, args); err != nil {
		logger.Warn("task_runner: %s")
		resp.Test.Error = errParseFailed.Error()
		return
	}
	if !args.Test {
		logger.Warnf("task_runner: %s", errTestDisabled.Error())
		resp.Test.Error = errTestDisabled.Error()
		return
	}
	logger.Info("task_runner: a log event for the Test task")
	resp.Test.Response = "test success"
}
