package main

import (
	"testing"

	"github.com/ooni/probe-cli/v3/internal/model/mocks"
)

func TestIndexLogger(t *testing.T) {
	t.Run("Debug", func(t *testing.T) {
		expected := "<0>antani"
		base := &mocks.Logger{
			MockDebug: func(message string) {
				if message != expected {
					t.Fatal("unexpected message")
				}
			},
		}
		logger := &indexLogger{
			indexstr: "<0>",
			logger:   base,
		}
		logger.Debug("antani")
	})

	t.Run("Info", func(t *testing.T) {
		expected := "<0>antani"
		base := &mocks.Logger{
			MockInfo: func(message string) {
				if message != expected {
					t.Fatal("unexpected message")
				}
			},
		}
		logger := &indexLogger{
			indexstr: "<0>",
			logger:   base,
		}
		logger.Info("antani")
	})

	t.Run("Warn", func(t *testing.T) {
		expected := "<0>antani"
		base := &mocks.Logger{
			MockWarn: func(message string) {
				if message != expected {
					t.Fatal("unexpected message")
				}
			},
		}
		logger := &indexLogger{
			indexstr: "<0>",
			logger:   base,
		}
		logger.Warn("antani")
	})

	t.Run("Debugf", func(t *testing.T) {
		expected := "<0>antani%d"
		base := &mocks.Logger{
			MockDebugf: func(format string, v ...any) {
				if format != expected {
					t.Fatal("unexpected message")
				}
			},
		}
		logger := &indexLogger{
			indexstr: "<0>",
			logger:   base,
		}
		logger.Debugf("antani%d", 11)
	})

	t.Run("Infof", func(t *testing.T) {
		expected := "<0>antani%d"
		base := &mocks.Logger{
			MockInfof: func(format string, v ...any) {
				if format != expected {
					t.Fatal("unexpected message")
				}
			},
		}
		logger := &indexLogger{
			indexstr: "<0>",
			logger:   base,
		}
		logger.Infof("antani%d", 11)
	})

	t.Run("Warnf", func(t *testing.T) {
		expected := "<0>antani%d"
		base := &mocks.Logger{
			MockWarnf: func(format string, v ...any) {
				if format != expected {
					t.Fatal("unexpected message")
				}
			},
		}
		logger := &indexLogger{
			indexstr: "<0>",
			logger:   base,
		}
		logger.Warnf("antani%d", 11)
	})
}
