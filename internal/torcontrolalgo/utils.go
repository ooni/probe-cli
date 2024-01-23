package torcontrolalgo

//
// utils.go - functions to encode/decode values for tor.
//
// SPDX-License-Identifier: MIT
//
// Adapted from https://github.com/cretz/bine.
//

import (
	"errors"
	"strings"
)

// partitionString returns the two parts of a string delimited by the first
// occurrence of ch. If ch does not exist, the second string is empty and the
// resulting bool is false. Otherwise it is true.
func partitionString(str string, ch byte) (string, string, bool) {
	index := strings.IndexByte(str, ch)
	if index < 0 {
		return str, "", false
	}
	return str[:index], str[index+1:], true
}

// unescapeSimpleQuotedStringIfNeeded calls unescapeSimpleQuotedString only if
// str is surrounded with double quotes.
func unescapeSimpleQuotedStringIfNeeded(str string) (string, error) {
	if len(str) >= 2 && str[0] == '"' && str[len(str)-1] == '"' {
		return unescapeSimpleQuotedString(str)
	}
	return str, nil
}

// ErrMissingQuotes mans that a supposedly-quoted string is missing quotes.
var ErrMissingQuotes = errors.New("torcontrol: missing quotes")

// unescapeSimpleQuotedString removes surrounding double quotes and then
// calls unescapeSimpleQuotedStringContents.
func unescapeSimpleQuotedString(str string) (string, error) {
	if len(str) < 2 || str[0] != '"' || str[len(str)-1] != '"' {
		return "", ErrMissingQuotes
	}
	return unescapeSimpleQuotedStringContents(str[1 : len(str)-1])
}

// ErrUnescapedQuote indicates that a quote char is not escaped as it ought to be.
var ErrUnescapedQuote = errors.New("torcontrol: unescaped quote")

// ErrUnescapedCROrLF indciates that a CR or LF is not escaped as it ought to be.
var ErrUnescapedCROrLF = errors.New("torcontrol: unescaped CR or LF")

// ErrUnexpectedEscape indicates we found an unexpected escape.
var ErrUnexpectedEscape = errors.New("torcontrol: unexpected escape")

// unescapeSimpleQuotedStringContents unescapes backslashes, double quotes,
// newlines, and carriage returns, or returns an error.
func unescapeSimpleQuotedStringContents(str string) (string, error) {
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
				return "", ErrUnescapedQuote
			}
			ret += "\""
			escaping = false

		case '\r', '\n':
			return "", ErrUnescapedCROrLF

		default:
			if escaping {
				if c == 'r' {
					ret += "\r"
				} else if c == 'n' {
					ret += "\n"
				} else {
					return "", ErrUnexpectedEscape
				}
			} else {
				ret += string(c)
			}
			escaping = false
		}
	}

	return ret, nil
}

// escapeSimpleQuotedStringIfNeeded calls escapeSimpleQuotedString only if the
// string contains a space, backslash, double quote, newline, or carriage return
// character.
func escapeSimpleQuotedStringIfNeeded(str string) string {
	if strings.ContainsAny(str, " \\\"\r\n") {
		return escapeSimpleQuotedString(str)
	}
	return str
}

var simpleQuotedStringEscapeReplacer = strings.NewReplacer(
	"\\", "\\\\",
	"\"", "\\\"",
	"\r", "\\r",
	"\n", "\\n",
)

// escapeSimpleQuotedString calls escapeSimpleQuotedStringContents and then
// surrounds the entire string with double quotes.
func escapeSimpleQuotedString(str string) string {
	return "\"" + escapeSimpleQuotedStringContents(str) + "\""
}

// escapeSimpleQuotedStringContents escapes backslashes, double quotes,
// newlines, and carriage returns in str.
func escapeSimpleQuotedStringContents(str string) string {
	return simpleQuotedStringEscapeReplacer.Replace(str)
}
