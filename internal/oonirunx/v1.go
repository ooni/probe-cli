package oonirunx

//
// OONI Run v1 implementation
//

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"

	"github.com/ooni/probe-cli/v3/internal/registryx"
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

// v1Measure performs a measurement using the given v1 OONI Run URL.
func v1Measure(ctx context.Context, config *LinkConfig, URL string) error {
	config.Session.Logger().Infof("oonirun/v1: running %s", URL)
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
		if pu.Path != "" && pu.Path != "/" {
			return ErrInvalidV1URLPath
		}
	default:
		return ErrInvalidV1URLScheme
	}
	name := pu.Query().Get("tn")
	if name == "" {
		return fmt.Errorf("%w: empty test name", ErrInvalidV1URLQueryArgument)
	}
	var inputs []string
	if ta := pu.Query().Get("ta"); ta != "" {
		inputs, err = v1ParseArguments(ta)
		if err != nil {
			return err
		}
	}
	if mv := pu.Query().Get("mv"); mv != "1.2.0" {
		return fmt.Errorf("%w: unknown minimum version", ErrInvalidV1URLQueryArgument)
	}
	factory := registryx.AllExperiments[name]
	args := make(map[string]any)
	extraOptions := make(map[string]any) // the v1 spec does not allow users to pass experiment options
	return factory.Oonirun(ctx, config.Session, inputs, args, extraOptions, config.DatabaseProps)
}

// v1ParseArguments parses the `ta` field of the query string.
func v1ParseArguments(ta string) ([]string, error) {
	var inputs []string
	pa, err := url.QueryUnescape(ta)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidV1URLQueryArgument, err.Error())
	}
	var arguments v1Arguments
	if err := json.Unmarshal([]byte(pa), &arguments); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidV1URLQueryArgument, err.Error())
	}
	inputs = arguments.URLs
	return inputs, nil
}
