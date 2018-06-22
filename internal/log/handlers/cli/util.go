package cli

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

// Finds the control character sequences (like colors)
var ctrlFinder = regexp.MustCompile("\x1b\x5b[0-9]+\x6d")

func escapeAwareRuneCountInString(s string) int {
	n := utf8.RuneCountInString(s)
	for _, sm := range ctrlFinder.FindAllString(s, -1) {
		n -= utf8.RuneCountInString(sm)
	}
	return n
}

func RightPad(str string, length int) string {
	return str + strings.Repeat(" ", length-escapeAwareRuneCountInString(str))
}
