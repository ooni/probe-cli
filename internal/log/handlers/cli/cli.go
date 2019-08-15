package cli

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/apex/log"
	"github.com/fatih/color"
	colorable "github.com/mattn/go-colorable"
	"github.com/ooni/probe-cli/internal/util"
)

// Default handler outputting to stderr.
var Default = New(os.Stdout)

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

func logSectionTitle(w io.Writer, f log.Fields) error {
	colWidth := 24

	title := f.Get("title").(string)
	fmt.Fprintf(w, "┏"+strings.Repeat("━", colWidth+2)+"┓\n")
	fmt.Fprintf(w, "┃ %s ┃\n", util.RightPad(title, colWidth))
	fmt.Fprintf(w, "┗"+strings.Repeat("━", colWidth+2)+"┛\n")
	return nil
}

func logTable(w io.Writer, f log.Fields) error {
	color := color.New(color.FgBlue)

	names := f.Names()

	var lines []string
	colWidth := 0
	for _, name := range names {
		if name == "type" {
			continue
		}
		line := fmt.Sprintf("%s: %s", color.Sprint(name), f.Get(name))
		lineLength := util.EscapeAwareRuneCountInString(line)
		lines = append(lines, line)
		if colWidth < lineLength {
			colWidth = lineLength
		}
	}

	fmt.Fprintf(w, "┏"+strings.Repeat("━", colWidth+2)+"┓\n")
	for _, line := range lines {
		fmt.Fprintf(w, "┃ %s ┃\n",
			util.RightPad(line, colWidth),
		)
	}
	fmt.Fprintf(w, "┗"+strings.Repeat("━", colWidth+2)+"┛\n")
	return nil
}

// TypedLog is used for handling special "typed" logs to the CLI
func (h *Handler) TypedLog(t string, e *log.Entry) error {
	switch t {
	case "engine":
		fmt.Fprintf(h.Writer, "[engine] %s\n", e.Message)
		return nil
	case "progress":
		perc := e.Fields.Get("percentage").(float64) * 100
		s := fmt.Sprintf("   %s %-25s",
			bold.Sprintf("%.2f%%", perc),
			e.Message)
		fmt.Fprint(h.Writer, s)
		fmt.Fprintln(h.Writer)
		return nil
	case "table":
		return logTable(h.Writer, e.Fields)
	case "measurement_item":
		return logMeasurementItem(h.Writer, e.Fields)
	case "measurement_summary":
		return logMeasurementSummary(h.Writer, e.Fields)
	case "result_item":
		return logResultItem(h.Writer, e.Fields)
	case "result_summary":
		return logResultSummary(h.Writer, e.Fields)
	case "section_title":
		return logSectionTitle(h.Writer, e.Fields)
	default:
		return h.DefaultLog(e)
	}
}

// DefaultLog is the default way of printing out logs
func (h *Handler) DefaultLog(e *log.Entry) error {
	color := Colors[e.Level]
	level := Strings[e.Level]
	names := e.Fields.Names()

	s := color.Sprintf("%s %-25s", bold.Sprintf("%*s", h.Padding+1, level), e.Message)
	for _, name := range names {
		if name == "source" {
			continue
		}
		s += fmt.Sprintf(" %s=%v", color.Sprint(name), e.Fields.Get(name))
	}

	fmt.Fprint(h.Writer, s)
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
