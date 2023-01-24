package dslx

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

func Round(err error) Func[float32, *Maybe[int]] {
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

func ToString() Func[int, *Maybe[string]] {
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
		pxFails     error
		expectedRes string
	}

	tests := []testCompose{
		{
			name:        "Compose2: pipeline succeeds, odd",
			pxFails:     nil,
			expectedRes: "27",
		},
		{
			name:        "Compose2: pipeline fails",
			pxFails:     errors.New("mocked"),
			expectedRes: "",
		},
	}
	var v float32 = 27.4
	for _, test := range tests {
		round := Round(test.pxFails)
		toString := ToString()
		composit := Compose2(round, toString)
		r := composit.Apply(context.Background(), v)
		if r.Error != test.pxFails {
			t.Fatalf("%s: expected error %s, got %s", test.name, test.pxFails, r.Error)
		}
		if r.State != test.expectedRes {
			t.Fatalf("%s: expected result state %v, got %v", test.name, test.expectedRes, r.State)
		}
	}

}

func TestCounter(t *testing.T) {
	type testCounter struct {
		name        string
		pxFails     error
		expectedRes int64
	}

	tests := []testCounter{
		{
			name:        "Counter: pipeline succeeds",
			pxFails:     nil,
			expectedRes: 1,
		},
		{
			name:        "Counter: pipeline fails",
			pxFails:     errors.New("mocked"),
			expectedRes: 0,
		},
	}
	var v float32 = 27.4
	for _, test := range tests {
		round := Round(test.pxFails)
		cnt := NewCounter[int]()
		composit := Compose2(round, cnt.Func())
		_ = composit.Apply(context.Background(), v)
		v := cnt.Value()
		if v != test.expectedRes {
			t.Fatalf("%s: expected counter value %v, got %v", test.name, test.expectedRes, v)
		}
	}
}
