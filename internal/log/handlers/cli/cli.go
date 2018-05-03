package cli

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/apex/log"
	"github.com/fatih/color"
	colorable "github.com/mattn/go-colorable"
)

// Default handler outputting to stderr.
var Default = New(os.Stderr)

// start time.
var start = time.Now()

var bold = color.New(color.Bold)

// Colors mapping.
var Colors = [...]*color.Color{
	log.DebugLevel: color.New(color.FgWhite),
	log.InfoLevel:  color.New(color.FgBlue),
	log.WarnLevel:  color.New(color.FgYellow),
	log.ErrorLevel: color.New(color.FgRed),
	log.FatalLevel: color.New(color.FgRed),
}

// Strings mapping.
var Strings = [...]string{
	log.DebugLevel: "•",
	log.InfoLevel:  "•",
	log.WarnLevel:  "•",
	log.ErrorLevel: "⨯",
	log.FatalLevel: "⨯",
}

// Handler implementation.
type Handler struct {
	mu      sync.Mutex
	Writer  io.Writer
	Padding int
}

// New handler.
func New(w io.Writer) *Handler {
	if f, ok := w.(*os.File); ok {
		return &Handler{
			Writer:  colorable.NewColorable(f),
			Padding: 3,
		}
	}

	return &Handler{
		Writer:  w,
		Padding: 3,
	}
}

// TypedLog is used for handling special "typed" logs to the CLI
func (h *Handler) TypedLog(t string, e *log.Entry) error {
	switch t {
	case "progress":
		// XXX replace this with something more fancy like https://github.com/tj/go-progress
		fmt.Fprintf(h.Writer, "%.1f%% [%s]: %s", e.Fields.Get("percentage").(float64)*100, e.Fields.Get("key"), e.Message)
		fmt.Fprintln(h.Writer)
		return nil
	case "result_item":
		return logResultItem(h.Writer, e.Fields)
	default:
		return h.DefaultLog(e)
	}
}

// DefaultLog is the default way of printing out logs
func (h *Handler) DefaultLog(e *log.Entry) error {
	color := Colors[e.Level]
	level := Strings[e.Level]
	names := e.Fields.Names()

	color.Fprintf(h.Writer, "%s %-25s", bold.Sprintf("%*s", h.Padding+1, level), e.Message)

	for _, name := range names {
		if name == "source" {
			continue
		}
		fmt.Fprintf(h.Writer, " %s=%s", color.Sprint(name), e.Fields.Get(name))
	}

	fmt.Fprintln(h.Writer)

	return nil
}

// HandleLog implements log.Handler.
func (h *Handler) HandleLog(e *log.Entry) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	t, isTyped := e.Fields["type"].(string)
	if isTyped {
		return h.TypedLog(t, e)
	}

	return h.DefaultLog(e)
}
