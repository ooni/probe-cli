package util

import (
	"testing"

	"github.com/fatih/color"
)

func TestEscapeAwareRuneCountInString(t *testing.T) {
	var bold = color.New(color.Bold)
	var myColor = color.New(color.FgBlue)

	s := myColor.Sprintf("•ABC%s%s", bold.Sprintf("DEF"), "\x1B[00;38;5;244m\x1B[m\x1B[00;38;5;33mGHI\x1B[0m")
	count := EscapeAwareRuneCountInString(s)
	if count != 10 {
		t.Errorf("Count was incorrect, got: %d, want: %d.", count, 10)
	}
}
