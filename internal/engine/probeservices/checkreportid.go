package probeservices

import (
	"context"
)

// CheckReportID checks whether the given ReportID exists.
func (c Client) CheckReportID(ctx context.Context, reportID string) (bool, error) {
	// Short circuit this API given that the ooni/api has also
	// done the same and so it's pointless to call.
	//
	// TODO(bassosimone): we should actually remove this method...
	return true, nil
}
