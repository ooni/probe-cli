package main

import (
	"path/filepath"
	"testing"
)

func TestGolangCheck(t *testing.T) {
	t.Run("successful case using the correct go version", func(t *testing.T) {
		golangCheck(filepath.Join("..", "..", "..", "GOVERSION"))
	})

	t.Run("invalid Go version where we expect a panic", func(t *testing.T) {
		var panicked bool
		func() {
			defer func() {
				panicked = recover() != nil
			}()
			golangCheck(filepath.Join("testdata", "GOVERSION"))
		}()
		if !panicked {
			t.Fatal("should have panicked")
		}
	})
}
