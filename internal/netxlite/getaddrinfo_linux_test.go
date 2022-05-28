//go:build cgo && linux

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
		name: "with C.EAI_SYSTEM and non-nil error",
		args: args{
			code: eaiSystem,
			err:  syscall.EAGAIN,
		},
		expects: expects{
			message: syscall.EAGAIN.Error(),
			code:    eaiSystem,
			err:     syscall.EAGAIN,
		},
	}, {
		name: "with C.EAI_SYSTEM and nil error",
		args: args{
			code: eaiSystem,
			err:  nil,
		},
		expects: expects{
			message: syscall.EMFILE.Error(),
			code:    eaiSystem,
			err:     syscall.EMFILE,
		},
	}, {
		name: "with C.EAI_NONAME",
		args: args{
			code: eaiNoName,
			err:  nil,
		},
		expects: expects{
			message: ErrOODNSNoSuchHost.Error(),
			code:    eaiNoName,
			err:     ErrOODNSNoSuchHost,
		},
	}, {
		name: "with an unhandled error",
		args: args{
			code: eaiBadFlags,
			err:  nil,
		},
		expects: expects{
			message: ErrOODNSMisbehaving.Error(),
			code:    eaiBadFlags,
			err:     ErrOODNSMisbehaving,
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
