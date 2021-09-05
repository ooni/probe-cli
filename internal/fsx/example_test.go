package fsx_test

import (
	"context"
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"syscall"

	"github.com/ooni/probe-cli/v3/internal/fsx"
	"github.com/ooni/probe-cli/v3/internal/netxlite/iox"
)

func ExampleOpenFile_openingDir() {
	filep, err := fsx.OpenFile("testdata")
	if !errors.Is(err, syscall.ENOENT) {
		log.Fatal("unexpected error", err)
	}
	if filep != nil {
		log.Fatal("expected nil fp")
	}
}

func ExampleOpenFile_openingFile() {
	filep, err := fsx.OpenFile(filepath.Join("testdata", "testfile.txt"))
	if err != nil {
		log.Fatal("unexpected error", err)
	}
	data, err := iox.ReadAllContext(context.Background(), filep)
	if err != nil {
		log.Fatal("unexpected error", err)
	}
	fmt.Printf("%d\n", len(data))
}
