package dslx

import (
	"context"
	"errors"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func getFn(err error) Func[int, *Maybe[int]] {
	return &fn{err: err}
}

type fn struct {
	err error
}

func (f *fn) Apply(ctx context.Context, i int) *Maybe[int] {
	return &Maybe[int]{
		Error: f.err,
		State: i + 1,
		Observations: []*Observations{
			{
				NetworkEvents: []*model.ArchivalNetworkEvent{{Tags: []string{"apply"}}},
			},
		},
	}
}

/*
Test cases:
- Compose 2 functions
  - pipeline succeeds
  - pipeline fails
*/
func TestCompose2(t *testing.T) {
	t.Run("Compose 2 functions", func(t *testing.T) {
		tests := map[string]struct {
			err    error
			input  int
			expect int
			numObs int
		}{
			"pipeline succeeds": {err: nil, input: 42, expect: 44, numObs: 2},
			"pipeline fails":    {err: errors.New("mocked"), input: 42, expect: 43, numObs: 1},
		}
		for name, tt := range tests {
			t.Run(name, func(t *testing.T) {
				f1 := getFn(tt.err)
				f2 := getFn(nil)
				composit := Compose2(f1, f2)
				r := composit.Apply(context.Background(), tt.input)
				if r.Error != tt.err {
					t.Fatalf("unexpected error")
				}
				if len(r.Observations) != tt.numObs {
					t.Fatalf("unexpected number of (merged) observations")
				}
			})
		}
	})
}

func TestGen(t *testing.T) {
	t.Run("Create composit of 14 functions", func(t *testing.T) {
		incFunc := getFn(nil)
		composit := Compose14(incFunc, incFunc, incFunc, incFunc, incFunc, incFunc, incFunc, incFunc,
			incFunc, incFunc, incFunc, incFunc, incFunc, incFunc)
		r := composit.Apply(context.Background(), 0)
		if r.Error != nil {
			t.Fatalf("unexpected error: %s", r.Error)
		}
		if r.State != 14 {
			t.Fatalf("unexpected result state")
		}
	})
}

func TestObservations(t *testing.T) {
	t.Run("Extract observations", func(t *testing.T) {
		fn1 := getFn(nil)
		fn2 := getFn(nil)
		composit := Compose2(fn1, fn2)
		r1 := composit.Apply(context.Background(), 3)
		r2 := composit.Apply(context.Background(), 42)
		if len(r1.Observations) != 2 || len(r2.Observations) != 2 {
			t.Fatalf("unexpected number of observations")
		}
		mergedObservations := ExtractObservations(r1, r2)
		if len(mergedObservations) != 4 {
			t.Fatalf("unexpected number of merged observations")
		}
	})
}

/*
Test cases:
- Success counter
  - pipeline succeeds
  - pipeline fails
*/
func TestCounter(t *testing.T) {
	t.Run("Success counter", func(t *testing.T) {
		tests := map[string]struct {
			err    error
			expect int64
		}{
			"pipeline succeeds": {err: nil, expect: 1},
			"pipeline fails":    {err: errors.New("mocked"), expect: 0},
		}
		for name, tt := range tests {
			t.Run(name, func(t *testing.T) {
				fn := getFn(tt.err)
				cnt := NewCounter[int]()
				composit := Compose2(fn, cnt.Func())
				_ = composit.Apply(context.Background(), 42)
				cntVal := cnt.Value()
				if cntVal != tt.expect {
					t.Fatalf("unexpected counter value")
				}
			})
		}
	})
}

/*
Test cases:
- Log errors
  - pipeline succeeds
  - pipeline fails
*/
func TestErrorLogger(t *testing.T) {
	t.Run("Log errors", func(t *testing.T) {
		tests := map[string]struct {
			err    error
			expect int
		}{
			"pipeline succeeds": {err: nil, expect: 0},
			"pipeline fails":    {err: errors.New("mocked"), expect: 1},
		}
		for name, tt := range tests {
			t.Run(name, func(t *testing.T) {
				fn := getFn(tt.err)
				errLog := &ErrorLogger{}
				fnWithErrors := RecordErrors(errLog, fn)
				fnWithErrors.Apply(context.Background(), 42)
				errs := errLog.Errors()
				if len(errs) != tt.expect {
					t.Fatalf("unexpected number of logged errors")
				}
				if len(errs) > 0 && errs[0] != tt.err {
					t.Fatalf("unexpected type of logged error")
				}
				if errLog.errors != nil {
					t.Fatalf("errors should be cleared after call to Errors")
				}
			})
		}
	})
}

/*
Test cases:
- Extract first error from list of *Maybe
  - without errors
  - with errors

- Extract first error excluding broken IPv6 errors
  - without errors
  - with errors
*/
func TestFirstError(t *testing.T) {
	networkUnreachable := errors.New(netxlite.FailureNetworkUnreachable)
	mockErr := errors.New("mocked")
	errRes := []*Maybe[string]{
		{Error: nil},
		{Error: networkUnreachable},
		{Error: mockErr},
	}
	noErrRes := []*Maybe[string]{
		{Error: nil},
		{Error: nil},
	}
	t.Run("Extract first error from list of *Maybe", func(t *testing.T) {
		t.Run("without errors", func(t *testing.T) {
			firstErr := FirstError(noErrRes...)
			if firstErr != nil {
				t.Fatalf("unexpected error: %s", firstErr)
			}
		})
		t.Run("with errors", func(t *testing.T) {
			firstErr := FirstError(errRes...)
			if firstErr != networkUnreachable {
				t.Fatalf("unexpected error: %s", firstErr)
			}
		})
	})
	t.Run("Extract first error excluding broken IPv6 errors", func(t *testing.T) {
		t.Run("without errors", func(t *testing.T) {
			firstErrExclIPv6 := FirstErrorExcludingBrokenIPv6Errors(noErrRes...)
			if firstErrExclIPv6 != nil {
				t.Fatalf("unexpected error: %s", firstErrExclIPv6)
			}
		})
		t.Run("with errors", func(t *testing.T) {
			firstErrExclIPv6 := FirstErrorExcludingBrokenIPv6Errors(errRes...)
			if firstErrExclIPv6 != mockErr {
				t.Fatalf("unexpected error: %s", firstErrExclIPv6)
			}
		})
	})
}
