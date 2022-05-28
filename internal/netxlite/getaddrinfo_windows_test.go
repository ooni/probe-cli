//go:build cgo && windows

package netxlite

import (
	"errors"
	"syscall"
	"testing"
)

func TestGetaddrinfoStateToError(t *testing.T) {
	type args struct {
		code int64
		err  error
	}
	type expects struct {
		message string // message obtained using .Error
		code    int64
		err     error
	}
	var inputs = []struct {
		name    string
		args    args
		expects expects
	}{{
		name: "with nonzero return code and error",
		args: args{
			code: int64(WSAHOST_NOT_FOUND),
			err:  syscall.EAGAIN,
		},
		expects: expects{
			message: syscall.EAGAIN.Error(),
			code:    int64(WSAHOST_NOT_FOUND),
			err:     syscall.EAGAIN,
		},
	}, {
		name: "with return code and nil error",
		args: args{
			code: int64(WSAHOST_NOT_FOUND),
			err:  nil,
		},
		expects: expects{
			message: WSAHOST_NOT_FOUND.Error(),
			code:    int64(WSAHOST_NOT_FOUND),
			err:     WSAHOST_NOT_FOUND,
		},
	}}
	for _, input := range inputs {
		t.Run(input.name, func(t *testing.T) {
			state := newGetaddrinfoState(getaddrinfoNumSlots)
			err := state.toError(input.args.code, input.args.err)
			if err == nil {
				t.Fatal("expected non-nil error here")
			}
			if err.Error() != input.expects.message {
				t.Fatal("unexpected error message")
			}
			var gaierr *ErrGetaddrinfo
			if !errors.As(err, &gaierr) {
				t.Fatal("cannot convert error to ErrGetaddrinfo")
			}
			if gaierr.Code != input.expects.code {
				t.Fatal("unexpected code")
			}
			if !errors.Is(gaierr.Underlying, input.expects.err) {
				t.Fatal("unexpected underlying error")
			}
		})
	}
}
