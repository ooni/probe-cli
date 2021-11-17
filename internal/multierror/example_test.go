package multierror_test

import (
	"errors"
	"fmt"

	"github.com/ooni/probe-cli/v3/internal/multierror"
)

func ExampleUnion() {
	root := errors.New("some error")
	me := multierror.New(root)
	check := errors.Is(me, root)
	fmt.Printf("%+v\n", check)
	// Output: true
}
