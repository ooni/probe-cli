package engine

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/engine/model"
)

func TestSubmitterNotEnabled(t *testing.T) {
	ctx := context.Background()
	submitter := NewSubmitter(ctx, SubmitterConfig{
		Enabled: false,
	})
	if _, ok := submitter.(stubSubmitter); !ok {
		t.Fatal("we did not get a stubSubmitter instance")
	}
	m := new(model.Measurement)
	if err := submitter.Submit(ctx, m); err != nil {
		t.Fatal(err)
	}
}

type FakeSubmitterSession struct {
	Calls uint32
	Error error
}

func (fs *FakeSubmitterSession) Submit(ctx context.Context, m *model.Measurement) error {
	atomic.AddUint32(&fs.Calls, 1)
	return fs.Error
}

var _ Submitter = &FakeSubmitterSession{}
var _ SubmitterSession = &FakeSubmitterSession{}

func TestNewSubmitterWithFailedSubmission(t *testing.T) {
	expected := errors.New("mocked error")
	ctx := context.Background()
	fakeSubmitter := &FakeSubmitterSession{Error: expected}
	submitter := NewSubmitter(ctx, SubmitterConfig{
		Enabled: true,
		Session: fakeSubmitter,
		Logger:  log.Log,
	})
	m := new(model.Measurement)
	err := submitter.Submit(context.Background(), m)
	if !errors.Is(err, expected) {
		t.Fatalf("not the error we expected: %+v", err)
	}
	if fakeSubmitter.Calls != 1 {
		t.Fatal("unexpected number of calls")
	}
}
