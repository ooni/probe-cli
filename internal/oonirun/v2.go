package oonirun

//
// OONI Run v2 implementation
//

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// v2Descriptor describes a single nettest to run.
type v2Descriptor struct {
	// NettestArguments contains the arguments for the nettest.
	NettestArguments v2Arguments `json:"ta"`

	// NettestName is the name of the nettest to run.
	NettestName string `json:"tn"`
}

// v2Arguments contains arguments for a given nettest.
type v2Arguments struct {
	// Inputs contains inputs for the experiment.
	Inputs []string `json:"inputs"`

	// Options contains the experiment options.
	Options map[string]string `json:"options"`
}

// ErrHTTPRequestFailed indicates that an HTTP request failed.
var ErrHTTPRequestFailed = errors.New("oonirun: HTTP request failed")

// getV2DescriptorsFromStaticURL GETs a list of V2Descriptor from
// a static URL (e.g., from a GitHub repo or from a Gist).
func getV2DescriptorsFromStaticURL(
	ctx context.Context, httpClient model.HTTPClient, URL string) ([]v2Descriptor, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", URL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, ErrHTTPRequestFailed
	}
	reader := io.LimitReader(resp.Body, 1<<22)
	data, err := netxlite.ReadAllContext(ctx, reader)
	if err != nil {
		return nil, err
	}
	var descs []v2Descriptor
	if err := json.Unmarshal(data, &descs); err != nil {
		return nil, err
	}
	return descs, nil
}

// v2MeasureStatic performs a measurement using a static (i.e., not served
// by the OONI API) v2 OONI Run URL and returns whether it failed.
func v2MeasureStatic(ctx context.Context, config *Config, URL string) error {
	clnt := config.Session.DefaultHTTPClient()
	descs, err := getV2DescriptorsFromStaticURL(ctx, clnt, URL)
	if err != nil {
		return err
	}
	logger := config.Session.Logger()
	for _, desc := range descs {
		if desc.NettestName == "" {
			logger.Warn("nettest name cannot be empty")
			continue
		}
		exp := &Experiment{
			Annotations:    config.Annotations,
			ExtraOptions:   desc.NettestArguments.Options,
			Inputs:         desc.NettestArguments.Inputs,
			InputFilePaths: nil,
			MaxRuntime:     config.MaxRuntime,
			Name:           desc.NettestName,
			NoCollector:    config.NoCollector,
			NoJSON:         config.NoJSON,
			Random:         config.Random,
			ReportFile:     config.ReportFile,
			Session:        config.Session,
		}
		if err := exp.Run(ctx); err != nil {
			logger.Warnf("cannot run experiment: %s", err.Error())
			continue
		}
	}
	return nil
}
