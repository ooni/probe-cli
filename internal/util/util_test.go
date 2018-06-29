package util

import (
	"testing"

	"github.com/fatih/color"
	ocolor "github.com/ooni/probe-cli/internal/colors"
)

func TestEscapeAwareRuneCountInString(t *testing.T) {
	var bold = color.New(color.Bold)
	var myColor = color.New(color.FgBlue)

	s := myColor.Sprintf("•ABC%s%s", bold.Sprintf("DEF"), ocolor.Red("GHI"))
	count := EscapeAwareRuneCountInString(s)
	if count != 10 {
		t.Errorf("Count was incorrect, got: %d, want: %d.", count, 10)
	}
}
