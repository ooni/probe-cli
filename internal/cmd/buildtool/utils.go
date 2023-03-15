package main

import (
	"errors"
	"fmt"
)

var errInvalidGOOSValue = errors.New("cannot build for runtime.GOOS value")

// generateLibrary generates the suitable library extension for the given GOOS.
func generateLibrary(prefix string, os string) (string, error) {
	switch os {
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
