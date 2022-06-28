package oonirun

//
// OONI Run v1 implementation
//

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
)

var (
	// ErrInvalidV1URLScheme indicates a v1 OONI Run URL has an invalid scheme.
	ErrInvalidV1URLScheme = errors.New("oonirun: invalid v1 URL scheme")

	// ErrInvalidV1URLHost indicates a v1 OONI Run URL has an invalid host.
	ErrInvalidV1URLHost = errors.New("oonirun: invalid v1 URL host")

	// ErrInvalidV1URLPath indicates a v1 OONI Run URL has an invalid path.
	ErrInvalidV1URLPath = errors.New("oonirun: invalid v1 URL path")

	// ErrInvalidV1URLQueryArgument indicates a v1 OONI Run URL query argument is invalid.
	ErrInvalidV1URLQueryArgument = errors.New("oonirun: invalid v1 URL query argument")
)

// v1Arguments contains arguments for a v1 OONI Run URL. These arguments are
// always encoded inside of the "ta" field, which is optional.
type v1Arguments struct {
	URLs []string `json:"urls"`
}

// v1Measure performs a measurement using a v1 OONI Run URL.
func v1Measure(ctx context.Context, config *Config, URL string) error {
	pu, err := url.Parse(URL)
	if err != nil {
		return err
	}
	switch pu.Scheme {
	case "https":
		if pu.Host != "run.ooni.io" {
			return ErrInvalidV1URLHost
		}
		if pu.Path != "/nettest" {
			return ErrInvalidV1URLPath
		}
	case "ooni":
		if pu.Host != "nettest" {
			return ErrInvalidV1URLHost
		}
		if pu.Path != "" {
			return ErrInvalidV1URLPath
		}
	default:
		return ErrInvalidV1URLScheme
	}
	name := pu.Query().Get("tn")
	if name == "" {
		return ErrInvalidV1URLQueryArgument
	}
	var inputs []string
	if ra := pu.Query().Get("ta"); ra != "" {
		pa, err := url.QueryUnescape(ra)
		if err != nil {
			return err
		}
		var arguments v1Arguments
		if err := json.Unmarshal([]byte(pa), &arguments); err != nil {
			return err
		}
		inputs = arguments.URLs
	}
	exp := &Experiment{
		Annotations:    config.Annotations,
		ExtraOptions:   nil, // no way to specify with v1 URLs
		Inputs:         inputs,
		InputFilePaths: nil,
		MaxRuntime:     config.MaxRuntime,
		Name:           name,
		NoCollector:    config.NoCollector,
		NoJSON:         config.NoJSON,
		Random:         config.Random,
		ReportFile:     config.ReportFile,
		Session:        config.Session,
	}
	return exp.Run(ctx)
}
