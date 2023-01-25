package stdlibx

//
// Wrappers for runtime
//

import (
	"fmt"
	"os"
)

// ExitOnError implements stdlib.
func (sp *stdlib) ExitOnError(err error, message string) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", message, err.Error())
		sp.exiter.Exit(1)
	}
}
