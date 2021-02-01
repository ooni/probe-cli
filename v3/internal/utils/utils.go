package utils

import (
	"bytes"
	"fmt"
	"os"
	"regexp"
	"strings"
	"unicode"
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

// RightPadd adds right padding in from of a string
func RightPad(str string, length int) string {
	c := length - EscapeAwareRuneCountInString(str)
	if c < 0 {
		c = 0
	}
	return str + strings.Repeat(" ", c)
}

// WrapString wraps the given string within lim width in characters.
//
// Wrapping is currently naive and only happens at white-space. A future
// version of the library will implement smarter wrapping. This means that
// pathological cases can dramatically reach past the limit, such as a very
// long word.
// This is taken from: https://github.com/mitchellh/go-wordwrap/tree/f253961a26562056904822f2a52d4692347db1bd
func WrapString(s string, lim uint) string {
	// Initialize a buffer with a slightly larger size to account for breaks
	init := make([]byte, 0, len(s))
	buf := bytes.NewBuffer(init)

	var current uint
	var wordBuf, spaceBuf bytes.Buffer

	for _, char := range s {
		if char == '\n' {
			if wordBuf.Len() == 0 {
				if current+uint(spaceBuf.Len()) > lim {
					current = 0
				} else {
					current += uint(spaceBuf.Len())
					spaceBuf.WriteTo(buf)
				}
				spaceBuf.Reset()
			} else {
				current += uint(spaceBuf.Len() + wordBuf.Len())
				spaceBuf.WriteTo(buf)
				spaceBuf.Reset()
				wordBuf.WriteTo(buf)
				wordBuf.Reset()
			}
			buf.WriteRune(char)
			current = 0
		} else if unicode.IsSpace(char) {
			if spaceBuf.Len() == 0 || wordBuf.Len() > 0 {
				current += uint(spaceBuf.Len() + wordBuf.Len())
				spaceBuf.WriteTo(buf)
				spaceBuf.Reset()
				wordBuf.WriteTo(buf)
				wordBuf.Reset()
			}

			spaceBuf.WriteRune(char)
		} else {

			wordBuf.WriteRune(char)

			if current+uint(spaceBuf.Len()+wordBuf.Len()) > lim && uint(wordBuf.Len()) < lim {
				buf.WriteRune('\n')
				current = 0
				spaceBuf.Reset()
			}
		}
	}

	if wordBuf.Len() == 0 {
		if current+uint(spaceBuf.Len()) <= lim {
			spaceBuf.WriteTo(buf)
		}
	} else {
		spaceBuf.WriteTo(buf)
		wordBuf.WriteTo(buf)
	}

	return buf.String()
}
