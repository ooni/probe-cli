package webconnectivityqa

import (
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netemx"
	"github.com/ooni/probe-cli/v3/internal/version"
)

// newMeasurement constructs a new [model.Measurement].
func newMeasurement(input string, measurer model.ExperimentMeasurer, t0 time.Time) *model.Measurement {
	return &model.Measurement{
		Annotations:               nil,
		DataFormatVersion:         "0.2.0",
		Extensions:                nil,
		ID:                        "",
		Input:                     model.MeasurementInput(input),
		InputHashes:               nil,
		MeasurementStartTime:      t0.Format(model.MeasurementDateFormat),
		MeasurementStartTimeSaved: t0,
		Options:                   []string{},
		ProbeASN:                  "AS137",
		ProbeCC:                   "IT",
		ProbeCity:                 "",
		ProbeIP:                   "127.0.0.1",
		ProbeNetworkName:          "Consortium GARR",
		ReportID:                  "",
		ResolverASN:               "AS137",
		ResolverIP:                netemx.ISPResolverAddress,
		ResolverNetworkName:       "Consortium GARR",
		SoftwareName:              "ooniprobe",
		SoftwareVersion:           version.Version,
		TestHelpers: map[string]any{
			"backend": map[string]string{
				"address": "https://0.th.ooni.org",
				"type":    "https",
			},
		},
		TestKeys:           nil,
		TestName:           measurer.ExperimentName(),
		MeasurementRuntime: 0,
		TestStartTime:      t0.Format(model.MeasurementDateFormat),
		TestVersion:        measurer.ExperimentVersion(),
	}
}
