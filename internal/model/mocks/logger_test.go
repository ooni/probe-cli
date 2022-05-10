package mocks

import "testing"

func TestLogger(t *testing.T) {
	t.Run("Debug", func(t *testing.T) {
		var called bool
		lo := &Logger{
			MockDebug: func(message string) {
				called = true
			},
		}
		lo.Debug("antani")
		if !called {
			t.Fatal("not called")
		}
	})

	t.Run("Debugf", func(t *testing.T) {
		var called bool
		lo := &Logger{
			MockDebugf: func(message string, v ...interface{}) {
				called = true
			},
		}
		lo.Debugf("antani", 1, 2, 3, 4)
		if !called {
			t.Fatal("not called")
		}
	})

	t.Run("Info", func(t *testing.T) {
		var called bool
		lo := &Logger{
			MockInfo: func(message string) {
				called = true
			},
		}
		lo.Info("antani")
		if !called {
			t.Fatal("not called")
		}
	})

	t.Run("Infof", func(t *testing.T) {
		var called bool
		lo := &Logger{
			MockInfof: func(message string, v ...interface{}) {
				called = true
			},
		}
		lo.Infof("antani", 1, 2, 3, 4)
		if !called {
			t.Fatal("not called")
		}
	})

	t.Run("Warn", func(t *testing.T) {
		var called bool
		lo := &Logger{
			MockWarn: func(message string) {
				called = true
			},
		}
		lo.Warn("antani")
		if !called {
			t.Fatal("not called")
		}
	})

	t.Run("Warnf", func(t *testing.T) {
		var called bool
		lo := &Logger{
			MockWarnf: func(message string, v ...interface{}) {
				called = true
			},
		}
		lo.Warnf("antani", 1, 2, 3, 4)
		if !called {
			t.Fatal("not called")
		}
	})
}
