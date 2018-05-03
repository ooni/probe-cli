package util

import (
	"fmt"
	"os"

	"github.com/ooni/probe-cli/internal/colors"
)

// Log outputs a log message.
func Log(msg string, v ...interface{}) {
	fmt.Printf("     %s\n", colors.Purple(fmt.Sprintf(msg, v...)))
}

// Fatal error
func Fatal(err error) {
	fmt.Fprintf(os.Stderr, "\n     %s %s\n\n", colors.Red("Error:"), err)
	os.Exit(1)
}
