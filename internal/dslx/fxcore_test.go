package dslx

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func getFn(err error, name string) Func[int, int] {
	return &fn{err: err, name: name}
}

type fn struct {
	err  error
	name string
}

func (f *fn) Apply(ctx context.Context, i *Maybe[int]) *Maybe[int] {
	if i.Error != nil {
		return i
	}
	return &Maybe[int]{
		Error: f.err,
		State: i.State + 1,
	}
}

func TestStageAdapter(t *testing.T) {
	t.Run("make sure that we handle a previous stage failure", func(t *testing.T) {
		unet := &mocks.UnderlyingNetwork{
			// explicitly empty so we crash if we try using underlying network functionality
		}
		netx := &netxlite.Netx{Underlying: unet}

		// create runtime
		rt := NewMinimalRuntime(model.DiscardLogger, time.Now(), MinimalRuntimeOptionMeasuringNetwork(netx))

		// create measurement pipeline where we run DNS lookups
		pipeline := DNSLookupGetaddrinfo(rt)

		// create input that contains an error
		input := &Maybe[*DomainToResolve]{
			Error: errors.New("mocked error"),
			State: nil,
		}

		// run the pipeline
		output := pipeline.Apply(context.Background(), input)

		// make sure the output contains the same error as the input
		if !errors.Is(output.Error, input.Error) {
			t.Fatal("unexpected error")
		}
	})
}

/*
Test cases:
- Compose 2 functions:
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
				f1 := getFn(tt.err, "maybe fail")
				f2 := getFn(nil, "succeed")
				composit := Compose2(f1, f2)
				r := composit.Apply(context.Background(), NewMaybeWithValue(tt.input))
				if r.Error != tt.err {
					t.Fatalf("unexpected error")
				}
			})
		}
	})
}

func TestGen(t *testing.T) {
	t.Run("Create composit of 14 functions", func(t *testing.T) {
		incFunc := getFn(nil, "succeed")
		composit := Compose14(incFunc, incFunc, incFunc, incFunc, incFunc, incFunc, incFunc, incFunc,
			incFunc, incFunc, incFunc, incFunc, incFunc, incFunc)
		r := composit.Apply(context.Background(), NewMaybeWithValue(0))
		if r.Error != nil {
			t.Fatalf("unexpected error: %s", r.Error)
		}
		if r.State != 14 {
			t.Fatalf("unexpected result state")
		}
	})
}
