package ddr

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/apex/log"
	"github.com/miekg/dns"
	"github.com/ooni/probe-cli/v3/internal/geoipx"
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
	Queries model.ArchivalDNSLookupResult `json:"queries"`

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

	resolver := ""
	if m.config.CustomResolver == nil {
		systemResolver := getSystemResolverAddress()
		if systemResolver == "" {
			return errors.New("could not get system resolver")
		}
		log.Infof("Using system resolver: %s", systemResolver)
		resolver = systemResolver
	} else {
		resolver = *m.config.CustomResolver
	}

	netx := &netxlite.Netx{}
	dialer := netx.NewDialerWithoutResolver(log.Log)
	transport := netxlite.NewUnwrappedDNSOverUDPTransport(dialer, resolver)
	encoder := &netxlite.DNSEncoderMiekg{}
	// As specified in RFC 9462 a DDR Query is a SVCB query for the _dns.resolver.arpa. domain
	query := encoder.Encode("_dns.resolver.arpa.", dns.TypeSVCB, true)
	t0 := time.Since(measurement.MeasurementStartTimeSaved).Seconds()

	resp, err := transport.RoundTrip(ctx, query)
	if err != nil {
		// Since we are using a custom transport, we need to check for context errors manually
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			failure := "interrupted"
			tk.Failure = &failure
			return nil
		}
		failure := err.Error()
		tk.Failure = &failure
		return nil
	}

	reply := &dns.Msg{}
	err = reply.Unpack(resp.Bytes())
	if err != nil {
		unpackError := err.Error()
		tk.Failure = &unpackError
		return nil
	}

	ddrResponse, err := decodeResponse(reply.Answer)
	if err != nil {
		decodingError := err.Error()
		tk.Failure = &decodingError
	}
	t := time.Since(measurement.MeasurementStartTimeSaved).Seconds()
	tk.Queries = createResult(t, t0, tk.Failure, resp, resolver, ddrResponse)
	tk.SupportsDDR = len(ddrResponse) > 0

	return nil
}

// decodeResponse decodes the response from the DNS query.
// DDR is only concerned with SVCB records, so we only decode those.
func decodeResponse(responseFields []dns.RR) ([]model.SVCBData, error) {
	responses := make([]model.SVCBData, 0)
	for _, rr := range responseFields {
		switch rr := rr.(type) {
		case *dns.SVCB:
			parsed, err := parseSvcb(rr)
			if err != nil {
				return nil, err
			}
			responses = append(responses, parsed)
		default:
			return nil, fmt.Errorf("unknown RR type: %T", rr)
		}
	}
	return responses, nil
}

func parseSvcb(rr *dns.SVCB) (model.SVCBData, error) {
	keys := make(map[string]string)
	for _, kv := range rr.Value {
		value := kv.String()
		key := kv.Key().String()
		keys[key] = value
	}

	return model.SVCBData{
		Priority:   rr.Priority,
		TargetName: rr.Target,
		Params:     keys,
	}, nil
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

func createResult(t float64, t0 float64, failure *string, resp model.DNSResponse, resolver string, svcbRecords []model.SVCBData) model.ArchivalDNSLookupResult {
	resolverHost, _, err := net.SplitHostPort(resolver)
	if err != nil {
		log.Warnf("Could not split resolver address %s: %s", resolver, err)
		resolverHost = resolver
	}
	asn, org, err := geoipx.LookupASN(resolverHost)
	if err != nil {
		log.Warnf("Could not lookup ASN for resolver %s: %s", resolverHost, err)
		asn = 0
		org = ""
	}

	answers := make([]model.ArchivalDNSAnswer, 0)
	for _, record := range svcbRecords {
		// Create an ArchivalDNSAnswer for each SVCB record
		// for this experiment, only the SVCB key is relevant.
		answers = append(answers, model.ArchivalDNSAnswer{
			ASN:        int64(asn),
			ASOrgName:  org,
			AnswerType: "SVCB",
			Hostname:   "",
			IPv4:       "",
			IPv6:       "",
			SVCB:       &record,
		})
	}

	return model.ArchivalDNSLookupResult{
		Answers:          answers,
		Engine:           "udp",
		Failure:          failure,
		GetaddrinfoError: 0,
		Hostname:         "_dns.resolver.arpa.",
		QueryType:        "SVCB",
		RawResponse:      resp.Bytes(),
		Rcode:            int64(resp.Rcode()),
		ResolverAddress:  resolverHost,
		T0:               t0,
		T:                t,
		Tags:             nil,
		TransactionID:    0,
	}
}

func NewExperimentMeasurer(config Config) model.ExperimentMeasurer {
	return &Measurer{
		config: config,
	}
}
