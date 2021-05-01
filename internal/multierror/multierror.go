// Package multierror contains code to manage multiple errors.
package multierror

import (
	"errors"
	"fmt"
	"strings"
)

// Union is the logical union of several errors. The Union will
// appear to be the Root error, except that it will actually
// be possible to look deeper and see specific child errors that
// occurred using errors.As and errors.Is.
type Union struct {
	// Children contains the underlying errors.
	Children []error

	// Root is the root error.
	Root error
}

// New creates a new Union error instance.
func New(root error) *Union {
	return &Union{Root: root}
}

// Unwrap returns the Root error of the Union error.
func (err Union) Unwrap() error {
	return err.Root
}

// Add adds the specified child error to the Union error.
func (err *Union) Add(child error) {
	err.Children = append(err.Children, child)
}

// AddWithPrefix adds the specified child error to the Union error
// with the specified prefix before the child error.
func (err *Union) AddWithPrefix(prefix string, child error) {
	err.Add(fmt.Errorf("%s: %w", prefix, child))
}

// Is returns true (1) if the err.Root error is target or (2) if
// any err.Children error is target.
func (err Union) Is(target error) bool {
	if errors.Is(err.Root, target) {
		return true
	}
	for _, c := range err.Children {
		if errors.Is(c, target) {
			return true
		}
	}
	return false
}

// Error returns a string representation of the Union error.
func (err Union) Error() string {
	var sb strings.Builder
	sb.WriteString(err.Root.Error())
	sb.WriteString(": [")
	for _, c := range err.Children {
		sb.WriteString(" ")
		sb.WriteString(c.Error())
		sb.WriteString(";")
	}
	sb.WriteString("]")
	return sb.String()
}
