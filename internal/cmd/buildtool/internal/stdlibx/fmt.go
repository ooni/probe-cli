package stdlibx

//
// Wrappers for fmt
//

import (
	"fmt"
	"io"
)

// MustFprintf implements Stdlib.
func (sp *stdlib) MustFprintf(w io.Writer, format string, v ...any) {
	_, err := fmt.Fprintf(w, format, v...)
	sp.ExitOnError(err, "fmt.Fprintf")
}
