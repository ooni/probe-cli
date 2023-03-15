package main

//
// Defines the CLI products we can build
//

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// product is a product we can build.
type product struct {
	// Pkg is the package to build
	Pkg string

	// Kind is the product kind.
	Kind string
}

// DestinationPath is the path where to build the product.
func (p *product) DestinationPath(goos, goarch string) string {
	suffix := ""
	if goos == "windows" {
		suffix = ".exe"
	}
	pv := strings.Split(p.Pkg, "/")
	runtimex.Assert(len(pv) >= 1, "expected at least one entry")
	pname := pv[len(pv)-1]
	return filepath.Join(p.Kind, fmt.Sprintf("%s-%s-%s%s", pname, goos, goarch, suffix))
}

// productMiniooni is the miniooni product.
var productMiniooni = &product{
	Pkg:  "./internal/cmd/miniooni",
	Kind: "CLI",
}

// productOoniprobe is the ooniprobe product.
var productOoniprobe = &product{
	Pkg:  "./cmd/ooniprobe",
	Kind: "CLI",
}

// productLibooniengine is the ooni engine shared library
var productLibooniengine = &product{
	Pkg:  "./internal/libooniengine",
	Kind: "lib",
}
