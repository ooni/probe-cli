package main

import (
	"errors"
	"fmt"
	"runtime"
)

var errInvalidGOOSValue = errors.New("cannot build for runtime.GOOS value")

// generateExtension generates the suitable library extension for the given GOOS.
func generateLibrary(prefix string) (string, error) {
	switch runtime.GOOS {
	case "windows":
		return fmt.Sprintf("%s.dll", prefix), nil
	case "linux":
		return fmt.Sprintf("%s.so", prefix), nil
	case "darwin":
		return fmt.Sprintf("%s.dylib", prefix), nil
	default:
		return "", errInvalidGOOSValue
	}
}
