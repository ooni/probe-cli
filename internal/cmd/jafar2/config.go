package main

import (
	"encoding/json"
	"io/ioutil"
)

// Config specifies what we should do.
type Config struct {
	// Blackhole contains a list of endpoints to blackhole.
	Blackhole []ConfigEndpoint

	// Download describes the constraints in the download path.
	Download *PathConstraints

	// Upload describes the constraints in the upload path.
	Upload *PathConstraints

	// Args contains the arguments to pass to miniooni.
	Args []string
}

// PathConstraints describes the constraints of a given path.
type PathConstraints struct {
	// Netem contains parameters for `tc qdisc ... netem ${parameters}`.
	Netem string

	// TBF contains parameters for `tc qdisc ... tbf ${parameters}`.
	TBF string
}

// ConfigEndpoint is an endpoint inside the spec.
type ConfigEndpoint struct {
	// Network is the endpoint network (e.g., "tcp").
	Network string

	// Address is the endpoint address (e.g., "1.1.1.1:443").
	Address string
}

// NewConfig reads the configuration from the given file.
func NewConfig(fname string) *Config {
	data, err := ioutil.ReadFile(fname)
	FatalOnError(err, "cannot open config file")
	var spec Config
	err = json.Unmarshal(data, &spec)
	FatalOnError(err, "cannot parse config file")
	return &spec
}
