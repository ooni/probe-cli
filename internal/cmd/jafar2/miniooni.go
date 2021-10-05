package main

import (
	"os"
)

// Miniooni refers to a temporary file holding a
// recompiled miniooni binary suitable for running
// inside our test container environment.
type Miniooni struct {
	fname string
}

// NewMiniooni creates a new Miniooni instance.
func NewMiniooni(shell Shell) *Miniooni {
	f := CreateFileTemp(".", "jafar2-miniooni")
	m := &Miniooni{fname: f.Name()}
	f.MustClose() // we don't need to keep it open
	cmd := NewCommandWithStdio(
		"go",
		"build",
		"-tags",
		"netgo",
		"-o",
		m.fname,
		"-v",
		"-ldflags",
		"-s -w -extldflags -static",
		"./internal/cmd/miniooni",
	)
	shell.MustRun(cmd)
	return m
}

// Path returns the path to the binary we've recompiled.
func (m *Miniooni) Path() string {
	return m.fname
}

// Cleanup removes the binary we've recompiled.
func (m *Miniooni) Cleanup() {
	os.Remove(m.fname)
}
