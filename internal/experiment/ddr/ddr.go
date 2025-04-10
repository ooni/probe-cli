package ddr

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/apex/log"
	"github.com/miekg/dns"
	"github.com/ooni/probe-cli/v3/internal/legacy/netx"
	"github.com/ooni/probe-cli/v3/internal/legacy/tracex"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

const (
	testName    = "ddr"
	testVersion = "0.1.0"
)

type Config struct {
	// CustomResolver is the custom resolver to use.
	// If empty, the system resolver is used.
	CustomResolver *string
}

// Measurer performs the measurement.
type Measurer struct {
	config Config
}

// ExperimentName implements ExperimentMeasurer.ExperimentName.
func (m *Measurer) ExperimentName() string {
	return testName
}

// ExperimentVersion implements ExperimentMeasurer.ExperimentVersion.
func (m *Measurer) ExperimentVersion() string {
	return testVersion
}

type TestKeys struct {
	// DNS Queries and results (as specified in https://github.com/ooni/spec/blob/master/data-formats/df-002-dnst.md#dns-data-format)
	Queries []model.ArchivalDNSLookupResult `json:"queries"`

	// SupportsDDR is true if DDR is supported.
	SupportsDDR bool `json:"supports_ddr"`

	// Failure is the failure that occurred, or nil.
	Failure *string `json:"failure"`
}

func (m *Measurer) Run(ctx context.Context, args *model.ExperimentArgs) error {
	log.SetLevel(log.DebugLevel)
	measurement := args.Measurement

	tk := &TestKeys{}
	measurement.TestKeys = tk

	timeout := 60 * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	resolverAddress := ""
	if m.config.CustomResolver == nil {
		systemResolver := getSystemResolverAddress()
		if systemResolver == "" {
			return errors.New("could not get system resolver")
		}
		log.Infof("Using system resolver: %s", systemResolver)
		resolverAddress = systemResolver
	} else {
		resolverAddress = *m.config.CustomResolver
	}

	netxlite := &netxlite.Netx{}
	evsaver := new(tracex.Saver)
	dialer := netxlite.NewDialerWithoutResolver(log.Log)
	baseResolver := netxlite.NewParallelUDPResolver(log.Log, dialer, resolverAddress)
	resolver := netx.NewResolver(netx.Config{
		BaseResolver: baseResolver,
		Saver:        evsaver,
	})

	// As specified in RFC 9462 a DDR Query is a SVCB query for the _dns.resolver.arpa. domain
	resp, err := resolver.LookupSVCB(ctx, "_dns.resolver.arpa.")
	if err != nil {
		tk.Failure = new(string)
		*tk.Failure = err.Error()
		return nil
	}
	queries := tracex.NewDNSQueriesList(measurement.MeasurementStartTimeSaved, evsaver.Read())

	for r := range resp {
		log.Debug(fmt.Sprintf("Got SVCB record: %v", r))
	}

	tk.Queries = queries
	tk.SupportsDDR = len(resp) > 0

	return nil
}

// Get the system resolver address from /etc/resolv.conf
// This should also be possible via querying the system resolver and checking the response
func getSystemResolverAddress() string {
	resolverConfig, err := dns.ClientConfigFromFile("/etc/resolv.conf")
	if err != nil {
		return ""
	}

	if len(resolverConfig.Servers) > 0 {
		return net.JoinHostPort(resolverConfig.Servers[0], resolverConfig.Port)
	}

	return ""
}

func NewExperimentMeasurer(config Config) model.ExperimentMeasurer {
	return &Measurer{
		config: config,
	}
}
