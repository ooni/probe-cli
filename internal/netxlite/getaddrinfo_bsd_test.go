//go:build cgo && (darwin || dragonfly || freebsd || openbsd)

package netxlite

import (
	"errors"
	"syscall"
	"testing"
)

func TestGetaddrinfoAIFlags(t *testing.T) {
	wrong := getaddrinfoAIFlags != (aiCanonname|aiV4Mapped|aiAll)&aiMask
	if wrong {
		t.Fatal("wrong flags for platform")
	}
}

func TestGetaddrinfoStateToError(t *testing.T) {
	type args struct {
		code int64
		err  error
		goos string
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
			goos: "darwin",
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
			goos: "darwin",
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
			goos: "darwin",
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
			goos: "darwin",
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
			err := state.toError(input.args.code, input.args.err, input.args.goos)
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
