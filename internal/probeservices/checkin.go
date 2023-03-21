package probeservices

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/checkincache"
	"github.com/ooni/probe-cli/v3/internal/checkintime"
	"github.com/ooni/probe-cli/v3/internal/httpapi"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/ooapi"
)

// CheckIn function is called by probes asking if there are tests to be run
// The config argument contains the mandatory settings.
// This function will additionally update the [checkincache] such that we
// track selected parts of the check-in API response.
// Returns the list of tests to run and the URLs, on success,
// or an explanatory error, in case of failure.
func (c Client) CheckIn(
	ctx context.Context, config model.OOAPICheckInConfig) (*model.OOAPICheckInResult, error) {
	// prepare endpoint and descriptor for the API call
	epnt := c.newHTTPAPIEndpoint()
	desc := ooapi.NewDescriptorCheckIn(&config)

	// issue the API call and handle failures
	resp, err := httpapi.Call(ctx, desc, epnt)
	if err != nil {
		return nil, err
	}

	// make sure we track selected parts of the response
	_ = checkincache.Store(c.KVStore, resp)

	// make sure we save the current time according to the check-in API
	checkintime.Save(resp.UTCTime)

	// emit warning if the probe clock is off
	checkintime.MaybeWarnAboutProbeClockBeingOff(c.Logger)

	return resp, nil
}
