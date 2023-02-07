package dslx

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func round(err error) Func[float32, *Maybe[int]] {
	return &roundFunc{err: err}
}

type roundFunc struct {
	err error
}

func (f *roundFunc) Apply(ctx context.Context, i float32) *Maybe[int] {
	return &Maybe[int]{
		Error: f.err,
		State: int(i),
		Observations: []*Observations{
			{
				NetworkEvents: []*model.ArchivalNetworkEvent{{Tags: []string{"round"}}},
			},
		},
	}
}

func toString() Func[int, *Maybe[string]] {
	return &toStringFunc{}
}

type toStringFunc struct{}

func (f *toStringFunc) Apply(ctx context.Context, i int) *Maybe[string] {
	return &Maybe[string]{
		Error: nil,
		State: fmt.Sprint(i),
		Observations: []*Observations{
			{
				NetworkEvents: []*model.ArchivalNetworkEvent{{Tags: []string{"toString"}}},
			},
		},
	}
}

func TestCompose2(t *testing.T) {
	type testCompose struct {
		name         string
		mockError    error
		expectedRes  string
		expectedObsN int
	}

	tests := []testCompose{
		{
			name:         "Compose2: pipeline succeeds",
			mockError:    nil,
			expectedRes:  "27",
			expectedObsN: 2,
		},
		{
			name:         "Compose2: pipeline fails",
			mockError:    errors.New("mocked"),
			expectedRes:  "",
			expectedObsN: 1,
		},
	}
	var v float32 = 27.4
	for _, test := range tests {
		funcRound := round(test.mockError)
		fungToString := toString()
		composit := Compose2(funcRound, fungToString)
		r := composit.Apply(context.Background(), v)
		if r.Error != test.mockError {
			t.Fatalf("%s: expected error %s, got %s", test.name, test.mockError, r.Error)
		}
		if r.State != test.expectedRes {
			t.Fatalf("%s: expected result state %v, got %v", test.name, test.expectedRes, r.State)
		}
		if len(r.Observations) != test.expectedObsN {
			t.Fatalf("%s: expected number of (merged) observations %d, got %d", test.name, test.expectedObsN, len(r.Observations))
		}
	}
}

func TestObservations(t *testing.T) {
	funcRound := round(nil)
	fungToString := toString()
	composit := Compose2(funcRound, fungToString)
	r1 := composit.Apply(context.Background(), 27.4)
	r2 := composit.Apply(context.Background(), 8.2)
	if len(r1.Observations) != 2 || len(r2.Observations) != 2 {
		t.Fatalf("Observations: unexpected number of observations")
	}
	mergedObservations := ExtractObservations(r1, r2)
	if len(mergedObservations) != 4 {
		t.Fatalf("Observations: expected number of merged observations %d, got %d", 4, len(mergedObservations))
	}
}

func TestCounter(t *testing.T) {
	type testCounter struct {
		name        string
		mockError   error
		expectedRes int64
	}

	tests := []testCounter{
		{
			name:        "Counter: pipeline succeeds",
			mockError:   nil,
			expectedRes: 1,
		},
		{
			name:        "Counter: pipeline fails",
			mockError:   errors.New("mocked"),
			expectedRes: 0,
		},
	}
	var v float32 = 27.4
	for _, test := range tests {
		funcRound := round(test.mockError)
		cnt := NewCounter[int]()
		composit := Compose2(funcRound, cnt.Func())
		_ = composit.Apply(context.Background(), v)
		cntVal := cnt.Value()
		if cntVal != test.expectedRes {
			t.Fatalf("%s: expected counter value %v, got %v", test.name, test.expectedRes, v)
		}
	}
}

func TestErrorLogger(t *testing.T) {
	type testErrorLogger struct {
		name        string
		mockError   error
		expectedRes int
	}

	tests := []testErrorLogger{
		{
			name:        "ErrorLogger: pipeline succeeds",
			mockError:   nil,
			expectedRes: 0,
		},
		{
			name:        "ErrorLogger: pipeline fails",
			mockError:   errors.New("mocked"),
			expectedRes: 1,
		},
	}
	var v float32 = 27.4
	for _, test := range tests {
		funcRound := round(test.mockError)
		errLog := &ErrorLogger{}
		funcRoundWithErrors := RecordErrors(errLog, funcRound)
		funcRoundWithErrors.Apply(context.Background(), v)
		errs := errLog.Errors()
		if len(errs) != test.expectedRes {
			t.Fatalf("%s: expected number of logged errors %v, got %v", test.name, test.expectedRes, len(errs))
		}
		if errLog.errors != nil {
			t.Fatalf("%s: expected reset errors, got non-nil list", test.name)
		}
		if len(errs) > 0 && errs[0] != test.mockError {
			t.Fatalf("%s: expected error %s, got %s", test.name, test.mockError, errs[0])

		}
	}
}

func TestFirstError(t *testing.T) {
	type testFirstError struct {
		mockResults []*Maybe[string]
		name        string
	}
	networkUnreachable := errors.New(netxlite.FailureNetworkUnreachable)
	mockErr := errors.New("mocked")
	// permutations: we want to test different orderings
	perm := [][]int{
		{0, 1, 2},
		{0, 2, 1},
		{1, 0, 2},
		{1, 2, 0},
		{2, 0, 1},
		{2, 1, 0},
	}
	res := []*Maybe[string]{
		{Error: nil},
		{Error: networkUnreachable},
		{Error: mockErr},
	}
	for _, p := range perm {
		var mockResults []*Maybe[string]
		for _, i := range p {
			mockResults = append(mockResults, res[i])
		}
		firstErrExclIPv6 := FirstErrorExcludingBrokenIPv6Errors(mockResults...)
		if firstErrExclIPv6 != mockErr {
			t.Fatalf("FirstErrorExcludingBrokenIPv6Errors: expected err %s, got %s, at perm %v", mockErr, firstErrExclIPv6, p)
		}
		firstErr := FirstError(mockResults...)
		expectedErr := res[p[0]].Error
		if expectedErr == nil {
			expectedErr = res[p[1]].Error
		}
		if firstErr != expectedErr {
			t.Fatalf("FirstError: expected err %s, got %s, at perm %v", expectedErr, firstErr, p)
		}
	}
	noErrRes := []*Maybe[string]{
		{Error: nil},
		{Error: nil},
	}
	firstErr := FirstError(noErrRes...)
	if firstErr != nil {
		t.Fatalf("FirstError: unexpected error %s", firstErr)
	}
	firstErr = FirstErrorExcludingBrokenIPv6Errors(noErrRes...)
	if firstErr != nil {
		t.Fatalf("FirstErrorExcludingBrokenIPv6Errors: unexpected error %s", firstErr)
	}
}

func inc() Func[int, *Maybe[int]] {
	return &incFunc{}
}

type incFunc struct{}

func (f *incFunc) Apply(ctx context.Context, i int) *Maybe[int] {
	return &Maybe[int]{State: i + 1}
}

func TestGen(t *testing.T) {
	incFunc := inc()
	composit := Compose14(incFunc, incFunc, incFunc, incFunc, incFunc, incFunc, incFunc, incFunc, incFunc, incFunc, incFunc, incFunc, incFunc, incFunc)
	r := composit.Apply(context.Background(), 0)
	if r.Error != nil {
		t.Fatalf("TestGen: unexpected error %s", r.Error)
	}
	if r.State != 14 {
		t.Fatalf("TestGen: expected result state %v, got %v", 14, r.State)
	}
}
