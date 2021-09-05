package iox_test

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/ooni/probe-cli/v3/internal/netxlite/iox"
)

func ExampleReadAllContext() {
	r := strings.NewReader("deadbeef")
	ctx := context.Background()
	out, err := iox.ReadAllContext(ctx, r)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%d\n", len(out))
	// Output: 8
}
