package netxlite

import (
	"context"
	"errors"
	"io"
	"testing"
)

func TestGetaddrinfoLookupANY(t *testing.T) {
	addrs, _, err := getaddrinfoLookupANY(context.Background(), "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	if len(addrs) != 1 || addrs[0] != "127.0.0.1" {
		t.Fatal("unexpected addrs", addrs)
	}
}

func TestErrorToGetaddrinfoRetval(t *testing.T) {
	type args struct {
		err error
	}
	tests := []struct {
		name string
		args args
		want int64
	}{{
		name: "with valid getaddrinfo error",
		args: args{
			newErrGetaddrinfo(144, nil),
		},
		want: 144,
	}, {
		name: "with another kind of error",
		args: args{io.EOF},
		want: 0,
	}, {
		name: "with nil error",
		args: args{nil},
		want: 0,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ErrorToGetaddrinfoRetvalOrZero(tt.args.err); got != tt.want {
				t.Errorf("ErrorToGetaddrinfoRetval() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_newErrGetaddrinfo(t *testing.T) {
	type args struct {
		code int64
		err  error
	}
	tests := []struct {
		name string
		args args
	}{{
		name: "common case",
		args: args{
			code: 17,
			err:  io.EOF,
		},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := newErrGetaddrinfo(tt.args.code, tt.args.err)
			if err == nil {
				t.Fatal("expected non-nil error")
			}
			if !errors.Is(err, tt.args.err) {
				t.Fatal("Unwrap() is not working correctly")
			}
			if err.Error() != tt.args.err.Error() {
				t.Fatal("Error() is not working correctly")
			}
			if err.Code != tt.args.code {
				t.Fatal("Code has not been copied correctly")
			}
		})
	}
}
