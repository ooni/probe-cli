//go:build cgo && linux

package netxlite

import (
	"errors"
	"runtime"
	"syscall"
	"testing"
)

func TestGetaddrinfoAIFlags(t *testing.T) {
	var wrong bool
	switch runtime.GOOS {
	case "android":
		wrong = getaddrinfoAIFlags != aiCanonname
	default:
		wrong = getaddrinfoAIFlags != (aiCanonname | aiV4Mapped | aiAll)
	}
	if wrong {
		t.Fatal("wrong flags for platform")
	}
}

func TestGetaddrinfoGetPlatformSpecificAiFlags(t *testing.T) {
	type args struct {
		goos string
	}
	type expects struct {
		flags int64
	}
	var inputs = []struct {
		name    string
		args    args
		expects expects
	}{{
		name: "using the Android platform",
		args: args{
			goos: "android",
		},
		expects: expects{
			flags: aiCanonname,
		},
	}, {
		name: "using Linux",
		args: args{
			goos: "linux",
		},
		expects: expects{
			flags: aiCanonname | aiV4Mapped | aiAll,
		},
	}, {
		name: "when the platform name is empty",
		args: args{
			goos: "",
		},
		expects: expects{
			flags: aiCanonname | aiV4Mapped | aiAll,
		},
	}}
	for _, input := range inputs {
		t.Run(input.name, func(t *testing.T) {
			flags := getaddrinfoGetPlatformSpecificAIFlags(input.args.goos)
			if int64(flags) != input.expects.flags {
				t.Fatal("invalid flags")
			}
		})
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
			goos: "linux",
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
			goos: "linux",
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
			goos: "linux",
		},
		expects: expects{
			message: ErrOODNSNoSuchHost.Error(),
			code:    eaiNoName,
			err:     ErrOODNSNoSuchHost,
		},
	}, {
		name: "with C.EAI_NODATA on Linux",
		args: args{
			code: eaiNoData,
			err:  nil,
			goos: "linux",
		},
		expects: expects{
			message: ErrOODNSNoAnswer.Error(),
			code:    eaiNoData,
			err:     ErrOODNSNoAnswer,
		},
	}, {
		name: "with C.EAI_NODATA on Android",
		args: args{
			code: eaiNoData,
			err:  nil,
			goos: "android",
		},
		expects: expects{
			message: ErrAndroidDNSCacheNoData.Error(),
			code:    eaiNoData,
			err:     ErrAndroidDNSCacheNoData,
		},
	}, {
		name: "with an unhandled error",
		args: args{
			code: eaiBadFlags,
			err:  nil,
			goos: "linux",
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
