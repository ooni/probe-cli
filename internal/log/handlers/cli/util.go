package cli

import (
	"strings"
	"unicode/utf8"
)

func RightPad(str string, length int) string {
	return str + strings.Repeat(" ", length-utf8.RuneCountInString(str))
}
