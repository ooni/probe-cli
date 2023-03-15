package main

import (
	"path/filepath"
	"testing"
)

func TestGolangCheck(t *testing.T) {
	// make sure the code does not panic when it runs
	golangCheck(filepath.Join("..", "..", "..", "GOVERSION"))
}
