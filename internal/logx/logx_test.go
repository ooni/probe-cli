package logx

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/apex/log"
	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
	"github.com/ooni/probe-cli/v3/internal/testingx"
)

func TestNewHandlerWithDefaultSettings(t *testing.T) {
	lh := NewHandlerWithDefaultSettings()
	if lh.Emoji {
		t.Fatal("expected false")
	}
	// Note: Go does not allow us to check whether lh.Now == time.Now
	if lh.StartTime.IsZero() {
		t.Fatal("expected non-zero time")
	}
	if lh.Writer != os.Stderr {
		t.Fatal("expected stderr")
	}
}

// creates a new handler with deterministic time to help with testing
func newHandlerForTesting() *Handler {
	lh := NewHandlerWithDefaultSettings()
	dtime := testingx.NewTimeDeterministic(time.Now())
	lh.Now = dtime.Now
	lh.StartTime = dtime.Now()
	return lh
}

func TestLogHandlerHandleLog(t *testing.T) {
	type config struct {
		// name of the test
		Name string

		// whether to use emojis
		Emoji bool

		// the verbosity level of the log entry
		Level log.Level

		// the string we expect for severity
		ExpectSeverity string
	}

	configs := []config{{
		Name:           "debug level without emoji",
		Emoji:          false,
		Level:          log.DebugLevel,
		ExpectSeverity: "<debug>",
	}, {
		Name:           "info level without emoji",
		Emoji:          false,
		Level:          log.InfoLevel,
		ExpectSeverity: "<info>",
	}, {
		Name:           "warn level without emoji",
		Emoji:          false,
		Level:          log.WarnLevel,
		ExpectSeverity: "<warn>",
	}, {
		Name:           "debug level with emoji",
		Emoji:          true,
		Level:          log.DebugLevel,
		ExpectSeverity: "🧐",
	}, {
		Name:           "info level with emoji",
		Emoji:          true,
		Level:          log.InfoLevel,
		ExpectSeverity: "  ",
	}, {
		Name:           "warn level with emoji",
		Emoji:          true,
		Level:          log.WarnLevel,
		ExpectSeverity: "🔥",
	}, {
		Name:           "fatal level with emoji",
		Emoji:          true,
		Level:          log.FatalLevel,
		ExpectSeverity: "🚨",
	}}

	for _, cnf := range configs {
		t.Run(cnf.Name, func(t *testing.T) {
			expected := fmt.Sprintf("[      1.000000] %s antani: map[error:EOF]\n", cnf.ExpectSeverity)
			var got string
			lh := newHandlerForTesting()
			lh.Emoji = cnf.Emoji
			lh.Writer = &mocks.Writer{
				MockWrite: func(b []byte) (int, error) {
					got = string(b)
					return len(b), nil
				},
			}
			lh.HandleLog(&log.Entry{
				Fields: map[string]any{
					"error": "EOF",
				},
				Level:   cnf.Level,
				Message: "antani",
			})
			if diff := cmp.Diff(expected, got); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
