// Package logx contains logging extensions
package logx

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/apex/log"
)

// Handler implements github.com/apex/log.Handler.
type Handler struct {
	// Emoji is OPTIONAL and indicates whether to enable emojis.
	Emoji bool

	// Now is the MANDATORY function to compute the current time.
	Now func() time.Time

	// StartTime is MANDATORY and indicates when we started logging.
	StartTime time.Time

	// Writer is MANDATORY and is the underlying writer.
	io.Writer
}

// NewHandlerWithDefaultSettings creates a new Handler with default settings.
func NewHandlerWithDefaultSettings() *Handler {
	return &Handler{
		Emoji:     false,
		Now:       time.Now,
		StartTime: time.Now(),
		Writer:    os.Stderr,
	}
}

var _ log.Handler = &Handler{}

// HandleLog implements log.Handler
func (h *Handler) HandleLog(e *log.Entry) (err error) {
	level := fmt.Sprintf("<%s>", e.Level.String())
	if h.Emoji {
		switch e.Level {
		case log.DebugLevel:
			level = "🧐"
		case log.InfoLevel:
			level = "  "
		case log.WarnLevel:
			level = "🔥"
		case log.FatalLevel:
			level = "🚨"
		default:
			// keep the original string
		}
	}
	elapsed := h.Now().Sub(h.StartTime)
	s := fmt.Sprintf("[%14.6f] %s %s", elapsed.Seconds(), level, e.Message)
	if len(e.Fields) > 0 {
		s += fmt.Sprintf(": %+v", e.Fields)
	}
	s += "\n"
	_, err = h.Writer.Write([]byte(s))
	return
}
