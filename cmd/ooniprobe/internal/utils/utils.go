package utils

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/fatih/color"
)

// Log outputs a log message.
func Log(msg string, v ...interface{}) {
	fmt.Printf("     %s\n", color.CyanString(msg, v...))
}

// Fatal error
func Fatal(err error) {
	fmt.Fprintf(os.Stderr, "\n     %s %s\n\n", color.RedString("Error:"), err)
	os.Exit(1)
}

// Finds the ansi escape sequences (like colors)
// Taken from: https://github.com/chalk/ansi-regex/blob/d9d806ecb45d899cf43408906a4440060c5c50e5/index.js
var ansiEscapes = regexp.MustCompile(`[\x1B\x9B][[\]()#;?]*` +
	`(?:(?:(?:[a-zA-Z\d]*(?:;[a-zA-Z\\d]*)*)?\x07)` +
	`|(?:(?:\d{1,4}(?:;\d{0,4})*)?[\dA-PRZcf-ntqry=><~]))`)

// EscapeAwareRuneCountInString counts the number of runes in a
// string taking into account escape sequences.
func EscapeAwareRuneCountInString(s string) int {
	n := utf8.RuneCountInString(s)
	for _, sm := range ansiEscapes.FindAllString(s, -1) {
		n -= utf8.RuneCountInString(sm)
	}
	return n
}

// RightPad adds right padding in from of a string
func RightPad(str string, length int) string {
	c := length - EscapeAwareRuneCountInString(str)
	if c < 0 {
		c = 0
	}
	return str + strings.Repeat(" ", c)
}
