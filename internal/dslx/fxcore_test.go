package dslx

import (
	"context"
	"errors"
	"fmt"
	"testing"

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
	}
}

func TestCompose2(t *testing.T) {
	type testCompose struct {
		name        string
		mockError   error
		expectedRes string
	}

	tests := []testCompose{
		{
			name:        "Compose2: pipeline succeeds, odd",
			mockError:   nil,
			expectedRes: "27",
		},
		{
			name:        "Compose2: pipeline fails",
			mockError:   errors.New("mocked"),
			expectedRes: "",
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
		fmt.Println(p)
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

}
