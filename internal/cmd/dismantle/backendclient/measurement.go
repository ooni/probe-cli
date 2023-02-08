package backendclient

import (
	"fmt"
	"runtime"
	"time"

	"github.com/ooni/probe-cli/v3/internal/geolocate"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/platform"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/version"
)

// dateFormat is the data format used to fill a measurement.
const dateFormat = "2006-01-02 15:04:05"

// NewMeasurement constructs a new measurement.
func NewMeasurement(
	location *geolocate.Results,
	testName string,
	testVersion string,
	testStartTime time.Time,
	reportID string,
	softwareName string,
	softwareVersion string,
	input string,
) *model.Measurement {
	utctimenow := time.Now().UTC()
	m := &model.Measurement{
		DataFormatVersion:         model.OOAPIReportDefaultDataFormatVersion,
		Input:                     model.MeasurementTarget(input),
		MeasurementStartTime:      utctimenow.Format(dateFormat),
		MeasurementStartTimeSaved: utctimenow,
		ProbeIP:                   model.DefaultProbeIP,
		ProbeASN:                  location.ProbeASNString(),
		ProbeCC:                   location.CountryCode,
		ProbeNetworkName:          location.NetworkName,
		ReportID:                  reportID,
		ResolverASN:               fmt.Sprintf("AS%d", location.ResolverASNumber), // XXX
		ResolverIP:                location.ResolverIPAddr,
		ResolverNetworkName:       location.ResolverASNetworkName,
		SoftwareName:              softwareName,
		SoftwareVersion:           softwareVersion,
		TestName:                  testName,
		TestStartTime:             testStartTime.Format(dateFormat),
		TestVersion:               testVersion,
	}
	m.AddAnnotation("architecture", runtime.GOARCH)
	m.AddAnnotation("engine_name", "ooniprobe-engine")
	m.AddAnnotation("engine_version", version.Version)
	m.AddAnnotation("go_version", runtimex.BuildInfo.GoVersion)
	m.AddAnnotation("platform", platform.Name())
	m.AddAnnotation("vcs_modified", runtimex.BuildInfo.VcsModified)
	m.AddAnnotation("vcs_revision", runtimex.BuildInfo.VcsRevision)
	m.AddAnnotation("vcs_time", runtimex.BuildInfo.VcsTime)
	m.AddAnnotation("vcs_tool", runtimex.BuildInfo.VcsTool)
	return m
}
