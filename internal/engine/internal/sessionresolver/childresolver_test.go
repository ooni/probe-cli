package sessionresolver

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
)

func TestTimeLimitedLookupSuccess(t *testing.T) {
	reso := &Resolver{}
	expect := []string{"8.8.8.8", "8.8.4.4"}
	re := &mocks.Resolver{
		MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
			return expect, nil
		},
	}
	ctx := context.Background()
	out, err := reso.timeLimitedLookup(ctx, re, "dns.google")
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(expect, out); diff != "" {
		t.Fatal(diff)
	}
}

func TestTimeLimitedLookupFailure(t *testing.T) {
	reso := &Resolver{}
	re := &mocks.Resolver{
		MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
			return nil, io.EOF
		},
	}
	ctx := context.Background()
	out, err := reso.timeLimitedLookup(ctx, re, "dns.google")
	if !errors.Is(err, io.EOF) {
		t.Fatal("not the error we expected", err)
	}
	if out != nil {
		t.Fatal("expected nil here")
	}
}

func TestTimeLimitedLookupWillTimeout(t *testing.T) {
	reso := &Resolver{}
	re := &mocks.Resolver{
		MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
			time.Sleep(time.Millisecond)
			return nil, io.EOF
		},
	}
	ctx := context.Background()
	out, err := reso.timeLimitedLookupx(ctx, time.Microsecond, re, "dns.google")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal("not the error we expected", err)
	}
	if out != nil {
		t.Fatal("expected nil here")
	}
}
