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
	expected := []string{"8.8.8.8", "8.8.4.4"}
	re := &mocks.Resolver{
		MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
			return expected, nil
		},
	}
	ctx := context.Background()
	out, err := timeLimitedLookup(ctx, re, "dns.google")
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(expected, out); diff != "" {
		t.Fatal(diff)
	}
}

func TestTimeLimitedLookupFailure(t *testing.T) {
	re := &mocks.Resolver{
		MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
			return nil, io.EOF
		},
	}
	ctx := context.Background()
	out, err := timeLimitedLookup(ctx, re, "dns.google")
	if !errors.Is(err, io.EOF) {
		t.Fatal("not the error we expected", err)
	}
	if out != nil {
		t.Fatal("expected nil here")
	}
}

func TestTimeLimitedLookupWillTimeout(t *testing.T) {
	done := make(chan bool)
	block := make(chan bool)
	re := &mocks.Resolver{
		MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
			defer close(done)
			<-block
			return nil, io.EOF
		},
	}
	ctx := context.Background()
	out, err := timeLimitedLookupWithTimeout(ctx, re, "dns.google", 10*time.Millisecond)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal("not the error we expected", err)
	}
	if out != nil {
		t.Fatal("expected nil here")
	}
	close(block)
	<-done
}
