package sessionresolver

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

type FakeResolver struct {
	Closed bool
	Data   []string
	Err    error
	Sleep  time.Duration
}

func (r *FakeResolver) LookupHost(ctx context.Context, hostname string) ([]string, error) {
	select {
	case <-time.After(r.Sleep):
		return r.Data, r.Err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (r *FakeResolver) CloseIdleConnections() {
	r.Closed = true
}

func TestTimeLimitedLookupSuccess(t *testing.T) {
	reso := &Resolver{}
	re := &FakeResolver{
		Data: []string{"8.8.8.8", "8.8.4.4"},
	}
	ctx := context.Background()
	out, err := reso.timeLimitedLookup(ctx, re, "dns.google")
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(re.Data, out); diff != "" {
		t.Fatal(diff)
	}
}

func TestTimeLimitedLookupFailure(t *testing.T) {
	reso := &Resolver{}
	re := &FakeResolver{
		Err: io.EOF,
	}
	ctx := context.Background()
	out, err := reso.timeLimitedLookup(ctx, re, "dns.google")
	if !errors.Is(err, re.Err) {
		t.Fatal("not the error we expected", err)
	}
	if out != nil {
		t.Fatal("expected nil here")
	}
}

func TestTimeLimitedLookupWillTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}
	reso := &Resolver{}
	re := &FakeResolver{
		Err:   io.EOF,
		Sleep: 20 * time.Second,
	}
	ctx := context.Background()
	out, err := reso.timeLimitedLookup(ctx, re, "dns.google")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal("not the error we expected", err)
	}
	if out != nil {
		t.Fatal("expected nil here")
	}
}
