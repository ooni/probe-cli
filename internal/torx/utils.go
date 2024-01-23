package torx

//
// utils.go - utilities to parse the control protocol.
//
// SPDX-License-Identifier: MIT
//
// Adapted from https://github.com/cretz/bine.
//

import (
	"errors"
	"strings"
)

// utilsPartitionString returns the two parts of a string delimited by the first
// occurrence of ch. If ch does not exist, the second string is empty and the
// resulting bool is false. Otherwise it is true.
func utilsPartitionString(str string, ch byte) (string, string, bool) {
	index := strings.IndexByte(str, ch)
	if index < 0 {
		return str, "", false
	}
	return str[:index], str[index+1:], true
}

// utilsUnescapeSimpleQuotedStringIfNeeded calls unescapeSimpleQuotedString only if
// str is surrounded with double quotes.
func utilsUnescapeSimpleQuotedStringIfNeeded(str string) (string, error) {
	if len(str) >= 2 && str[0] == '"' && str[len(str)-1] == '"' {
		return utilsUnescapeSimpleQuotedString(str)
	}
	return str, nil
}

// ErrControlMissingQuotes mans that a supposedly-quoted string is missing quotes.
var ErrControlMissingQuotes = errors.New("torx: control: missing quotes")

// utilsUnescapeSimpleQuotedString removes surrounding double quotes and then
// calls unescapeSimpleQuotedStringContents.
func utilsUnescapeSimpleQuotedString(str string) (string, error) {
	if len(str) < 2 || str[0] != '"' || str[len(str)-1] != '"' {
		return "", ErrControlMissingQuotes
	}
	return utilsUnescapeSimpleQuotedStringContents(str[1 : len(str)-1])
}

// ErrControlUnescapedQuote indicates that a quote char is not escaped as it ought to be.
var ErrControlUnescapedQuote = errors.New("torx: control: unescaped quote")

// ErrControlUnescapedCROrLF indciates that a CR or LF is not escaped as it ought to be.
var ErrControlUnescapedCROrLF = errors.New("torx: control: unescaped CR or LF")

// ErrControlUnexpectedEscape indicates we found an unexpected escape.
var ErrControlUnexpectedEscape = errors.New("torx: control: unexpected escape")

// utilsUnescapeSimpleQuotedStringContents unescapes backslashes, double quotes,
// newlines, and carriage returns, or returns an error.
func utilsUnescapeSimpleQuotedStringContents(str string) (string, error) {
	// TODO(bassosimone): we should refactor this function to use
	// a string builder rather than appending to a string.
	ret := ""
	escaping := false
	for _, c := range str {
		switch c {
		case '\\':
			if escaping {
				ret += "\\"
			}
			escaping = !escaping

		case '"':
			if !escaping {
				return "", ErrControlUnescapedQuote
			}
			ret += "\""
			escaping = false

		case '\r', '\n':
			return "", ErrControlUnescapedCROrLF

		default:
			if escaping {
				if c == 'r' {
					ret += "\r"
				} else if c == 'n' {
					ret += "\n"
				} else {
					return "", ErrControlUnexpectedEscape
				}
			} else {
				ret += string(c)
			}
			escaping = false
		}
	}

	return ret, nil
}

// utilsEscapeSimpleQuotedStringIfNeeded calls escapeSimpleQuotedString only if the
// string contains a space, backslash, double quote, newline, or carriage return
// character.
func utilsEscapeSimpleQuotedStringIfNeeded(str string) string {
	if strings.ContainsAny(str, " \\\"\r\n") {
		return utilsEscapeSimpleQuotedString(str)
	}
	return str
}

var utilsSimpleQuotedStringEscapeReplacer = strings.NewReplacer(
	"\\", "\\\\",
	"\"", "\\\"",
	"\r", "\\r",
	"\n", "\\n",
)

// utilsEscapeSimpleQuotedString calls escapeSimpleQuotedStringContents and then
// surrounds the entire string with double quotes.
func utilsEscapeSimpleQuotedString(str string) string {
	return "\"" + utilsEscapeSimpleQuotedStringContents(str) + "\""
}

// utilsEscapeSimpleQuotedStringContents escapes backslashes, double quotes,
// newlines, and carriage returns in str.
func utilsEscapeSimpleQuotedStringContents(str string) string {
	return utilsSimpleQuotedStringEscapeReplacer.Replace(str)
}
