package multierror_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/engine/internal/multierror"
)

func TestEmpty(t *testing.T) {
	root := errors.New("antani")
	var err error = multierror.New(root)
	if err.Error() != "antani: []" {
		t.Fatal("unexpected Error value")
	}
	if !errors.Is(err, root) {
		t.Fatal("error should be root")
	}
	if !errors.Is(errors.Unwrap(err), root) {
		t.Fatal("unwrapping did not return root")
	}
	if errors.Is(err, io.EOF) {
		t.Fatal("error should not be EOF")
	}
}

func TestNonEmpty(t *testing.T) {
	root := errors.New("antani")
	container := multierror.New(root)
	container.AddWithPrefix("first operation failed", io.EOF)
	container.AddWithPrefix("second operation failed", context.Canceled)
	var err error = container
	expect := "antani: [ first operation failed: EOF; second operation failed: context canceled;]"
	if diff := cmp.Diff(err.Error(), expect); diff != "" {
		t.Fatal(diff)
	}
	if !errors.Is(err, root) {
		t.Fatal("error should be root")
	}
	if !errors.Is(errors.Unwrap(err), root) {
		t.Fatal("unwrapping did not return root")
	}
	if !errors.Is(err, io.EOF) {
		t.Fatal("error should be EOF")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatal("error should be context.Canceled")
	}
	var as *multierror.Union
	if !errors.As(err, &as) {
		t.Fatal("cannot cast error to multierror.Union")
	}
	if !errors.Is(as.Root, root) {
		t.Fatal("unexpected root")
	}
	if len(as.Children) != 2 {
		t.Fatal("unexpected number of children")
	}
}

type SpecificRootError struct {
	Value int
}

func (sre SpecificRootError) Error() string {
	return fmt.Sprintf("%d", sre.Value)
}

func TestAsWorksForRoot(t *testing.T) {
	const expected = 144
	var (
		err error = multierror.New(&SpecificRootError{Value: expected})
		sre *SpecificRootError
	)
	if !errors.As(err, &sre) {
		t.Fatal("cannot cast error to original type")
	}
	if sre.Value != expected {
		t.Fatal("unexpected sre.Value")
	}
}
