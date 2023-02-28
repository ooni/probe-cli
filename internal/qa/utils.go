package qa

import (
	"errors"
	"math"
	"net"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/version"
)

// probeIP returns the IP that the probe should use. This function
// will panic after 65535 repetitions when it has exhausted the
// whole address space. The zero value is ready to use.
type probeIP struct {
	// third is the third octet
	third uint8

	// fourth is the fourth octet
	fourth uint8
}

// Next returns the next suitable probe IP.
func (p *probeIP) Next() string {
	third := p.third
	fourth := p.fourth
	if p.fourth < math.MaxUint8 {
		p.fourth++
	} else if p.third < math.MaxUint8 {
		p.fourth = 0
		p.third++
	} else {
		panic(errors.New("qa: out of all available IP addresses"))
	}
	return string(net.IPv4(130, 192, third, fourth))
}

// newMeasurement creates a new fake [model.Measurement].
func newMeasurement(testName, testVersion string) *model.Measurement {
	utctimenow := time.Now().UTC()
	return &model.Measurement{
		Annotations:               map[string]string{},
		DataFormatVersion:         "0.2.0",
		Extensions:                map[string]int64{},
		ID:                        "",
		Input:                     "",
		InputHashes:               []string{},
		MeasurementStartTime:      utctimenow.Format(model.MeasurementDateFormat),
		MeasurementStartTimeSaved: utctimenow,
		Options:                   []string{},
		ProbeASN:                  "AS137",
		ProbeCC:                   "IT",
		ProbeCity:                 "",
		ProbeIP:                   model.DefaultProbeIP,
		ProbeNetworkName:          "Consortium GARR",
		ReportID:                  "",
		ResolverASN:               "AS137",
		ResolverIP:                "130.192.3.24",
		ResolverNetworkName:       "Consortium GARR",
		SoftwareName:              "miniooni",
		SoftwareVersion:           version.Version,
		TestHelpers:               nil,
		TestKeys:                  nil,
		TestName:                  testName,
		MeasurementRuntime:        0,
		TestStartTime:             utctimenow.Format(model.MeasurementDateFormat),
		TestVersion:               testVersion,
	}
}
