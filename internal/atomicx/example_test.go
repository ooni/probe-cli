package atomicx_test

import (
	"fmt"

	"github.com/ooni/probe-cli/v3/internal/atomicx"
)

func Example_typicalUsage() {
	v := &atomicx.Int64{}
	v.Add(1)
	fmt.Printf("%d\n", v.Load())
	// Output: 1
}
