package probeservices

import (
	"context"
	"errors"

	"github.com/ooni/probe-cli/v3/internal/httpclientx"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/urlx"
)

// ErrNoMatchingPolicy
var ErrNoMatchingPolicy = errors.New("no submission policy matches this probe")

// GetManifest queries the manifest API and returns the parsed manifest
func (c Client) GetManifest(ctx context.Context) (*model.OOAPIManifest, error) {
	URL, err := urlx.ResolveReference(c.BaseURL, "/api/v1/manifest", "")
	if err != nil {
		return nil, err
	}

	return httpclientx.GetJSON[*model.OOAPIManifest](
		ctx,
		httpclientx.NewEndpoint(URL).WithHostOverride(c.Host),
		&httpclientx.Config{
			Client:    c.HTTPClient,
			Logger:    c.Logger,
			UserAgent: c.UserAgent,
		})
}

// matchField reports whether a manifest match value matches a probe value.
func matchField(pattern, value string) bool {
	return pattern == "*" || pattern == value
}

// GetRangesFromPolicy returns the age range and the minimum measurement count
// from the first submission policy in the manifest whose match clause matches
// the given probeCC/probeASN.
func GetRangesFromPolicy(
	manifest *model.OOAPIManifest,
	probeCC, probeASN string,
) (ageRange [2]uint32, minMeasurementCount uint32, err error) {
	for _, item := range manifest.Manifest.SubmissionPolicy {
		if !matchField(item.Match.ProbeCC, probeCC) || !matchField(item.Match.ProbeASN, probeASN) {
			continue
		}
		if len(item.Policy.Age) < 2 {
			return ageRange, 0, errors.New("probeservices: submission policy has an invalid age range")
		}
		return [2]uint32{item.Policy.Age[0], item.Policy.Age[1]}, item.Policy.MinMeasurementCount, nil
	}
	return ageRange, 0, ErrNoMatchingPolicy
}
