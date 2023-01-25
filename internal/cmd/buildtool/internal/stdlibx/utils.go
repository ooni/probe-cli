package stdlibx

//
// Utilities
//

import (
	"bytes"

	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// mustReadFirstLine reads the first line from a slice of bytes.
func mustReadFirstLine(data []byte) string {
	vec := bytes.Split(data, []byte("\n"))
	runtimex.Assert(len(vec) >= 1, "expected at least one line")
	return string(vec[0])
}
