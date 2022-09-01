package measurexlite

import (
	"fmt"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model/mocks"
)

func TestNewOperationLogger(t *testing.T) {
	t.Run("for short operation and no error", func(t *testing.T) {
		var (
			lines []string
			mu    sync.Mutex
		)
		logger := &mocks.Logger{
			MockInfof: func(format string, v ...interface{}) {
				line := fmt.Sprintf(format, v...)
				mu.Lock()
				lines = append(lines, line)
				mu.Unlock()
			},
		}
		ol := NewOperationLogger(logger, "antani%d", 0)
		ol.Stop(nil)
		if len(lines) != 1 {
			t.Fatal("unexpected number of lines")
		}
		if lines[0] != "antani0... ok" {
			t.Fatal("unexpected first line", lines[0])
		}
	})

	t.Run("for short operation and error", func(t *testing.T) {
		var (
			lines []string
			mu    sync.Mutex
		)
		logger := &mocks.Logger{
			MockInfof: func(format string, v ...interface{}) {
				line := fmt.Sprintf(format, v...)
				mu.Lock()
				lines = append(lines, line)
				mu.Unlock()
			},
		}
		ol := NewOperationLogger(logger, "antani%d", 0)
		ol.Stop(io.EOF)
		if len(lines) != 1 {
			t.Fatal("unexpected number of lines")
		}
		if lines[0] != "antani0... EOF" {
			t.Fatal("unexpected first line", lines[0])
		}
	})

	t.Run("for longer operation and no error", func(t *testing.T) {
		var (
			lines []string
			mu    sync.Mutex
		)
		logger := &mocks.Logger{
			MockInfof: func(format string, v ...interface{}) {
				line := fmt.Sprintf(format, v...)
				mu.Lock()
				lines = append(lines, line)
				mu.Unlock()
			},
		}
		const maxwait = 100 * time.Microsecond
		ol := newOperationLogger(maxwait, logger, "antani%d", 0)
		ol.wg.Wait() // wait for the message to be emitted
		ol.Stop(nil)
		if len(lines) != 2 {
			t.Fatal("unexpected number of lines")
		}
		if lines[0] != "antani0... in progress" {
			t.Fatal("unexpected first line", lines[0])
		}
		if lines[1] != "antani0... ok" {
			t.Fatal("unexpected first line", lines[0])
		}
	})

	t.Run("for longer operation and error", func(t *testing.T) {
		var (
			lines []string
			mu    sync.Mutex
		)
		logger := &mocks.Logger{
			MockInfof: func(format string, v ...interface{}) {
				line := fmt.Sprintf(format, v...)
				mu.Lock()
				lines = append(lines, line)
				mu.Unlock()
			},
		}
		const maxwait = 100 * time.Microsecond
		ol := newOperationLogger(maxwait, logger, "antani%d", 0)
		ol.wg.Wait() // wait for the message to be emitted
		ol.Stop(io.EOF)
		if len(lines) != 2 {
			t.Fatal("unexpected number of lines")
		}
		if lines[0] != "antani0... in progress" {
			t.Fatal("unexpected first line", lines[0])
		}
		if lines[1] != "antani0... EOF" {
			t.Fatal("unexpected first line", lines[0])
		}
	})
}
