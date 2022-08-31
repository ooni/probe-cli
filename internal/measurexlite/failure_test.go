package measurexlite

import (
	"errors"
	"io"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func TestNewFailure(t *testing.T) {
	type args struct {
		err error
	}
	tests := []struct {
		name string
		args args
		want *string
	}{{
		name: "when error is nil",
		args: args{
			err: nil,
		},
		want: nil,
	}, {
		name: "when error is wrapped and failure meaningful",
		args: args{
			err: &netxlite.ErrWrapper{
				Failure: netxlite.FailureConnectionRefused,
			},
		},
		want: func() *string {
			s := netxlite.FailureConnectionRefused
			return &s
		}(),
	}, {
		name: "when error is wrapped and failure is not meaningful",
		args: args{
			err: &netxlite.ErrWrapper{},
		},
		want: func() *string {
			s := "unknown_failure: errWrapper.Failure is empty"
			return &s
		}(),
	}, {
		name: "when error is not wrapped but wrappable",
		args: args{err: io.EOF},
		want: func() *string {
			s := "eof_error"
			return &s
		}(),
	}, {
		name: "when the error is not wrapped and not wrappable",
		args: args{
			err: errors.New("use of closed socket 127.0.0.1:8080->10.0.0.1:22"),
		},
		want: func() *string {
			s := "unknown_failure: use of closed socket [scrubbed]->[scrubbed]"
			return &s
		}(),
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewFailure(tt.args.err)
			if tt.want == nil && got == nil {
				return
			}
			if tt.want == nil && got != nil {
				t.Errorf("NewFailure:  want %+v, got %s", tt.want, *got)
				return
			}
			if tt.want != nil && got == nil {
				t.Errorf("NewFailure:  want %s, got %+v", *tt.want, got)
				return
			}
			if *tt.want != *got {
				t.Errorf("NewFailure:  want %s, got %s", *tt.want, *got)
				return
			}
		})
	}
}
