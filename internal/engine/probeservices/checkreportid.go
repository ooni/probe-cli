package probeservices

import "context"

// CheckReportID checks whether the given ReportID exists.
func (c Client) CheckReportID(ctx context.Context, reportID string) (bool, error) {
	// The API has been returning true for some time now. So, it does not make
	// sense for us to actually issue the API call. Let's short circuit it.
	//
	// See https://github.com/ooni/api/blob/80913ffd446e7a46761c4c8fdf3e42174f0ce645/newapi/ooniapi/private.py#L208
	//
	// TODO(https://github.com/ooni/probe/issues/2389): remove this code.
	return true, nil
}
