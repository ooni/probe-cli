package main

import (
	"fmt"
	"os"
)

// Trampoline is the docker trampoline script.
type Trampoline struct {
	file string
}

// NewTrampoline creates a new Trampoline instance.
func NewTrampoline(config *Config, miniooni *Miniooni) *Trampoline {
	fp := CreateFileTemp(".", "jafar2-trampoline")
	t := &Trampoline{file: fp.Name()}
	t.write(fp, config, miniooni)
	fp.MustClose()
	err := os.Chmod(t.file, 0755)
	FatalOnError(err, "cannot make trampoline executable")
	return t
}

// Path returns the path of the trampoline script.
func (t *Trampoline) Path() string {
	return t.file
}

// Cleanup cleanups the trampoline script.
func (t *Trampoline) Cleanup() {
	os.Remove(t.file)
}

func (t *Trampoline) write(f *File, c *Config, m *Miniooni) {
	f.WriteString("#!/bin/sh\n")
	f.WriteString("set -ex\n")
	if c.Upload != nil {
		f.WriteString(fmt.Sprintf(
			"tc qdisc add dev eth0 root handle 1: netem %s\n",
			c.Upload.Netem,
		))
		f.WriteString(fmt.Sprintf(
			"tc qdisc add dev eth0 parent 1: handle 2: tbf %s\n",
			c.Upload.TBF,
		))
	}
	f.WriteString(fmt.Sprintf("%s %s\n", m.Path(), QuoteShellArgs(c.Args)))
}
