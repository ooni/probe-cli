package ddr

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/apex/log"
	"github.com/miekg/dns"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

const (
	testName    = "ddr"
	testVersion = "0.1.0"
)

type Config struct {
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

type DDRResponse struct {
	Priority int               `json:"priority"`
	Target   string            `json:"target"`
	Keys     map[string]string `json:"keys"`
}

type TestKeys struct {
	// DDRResponse is the DDR response.
	DDRResponse []DDRResponse `json:"ddr_responses"`

	// SupportsDDR is true if DDR is supported.
	SupportsDDR bool `json:"supports_ddr"`

	// Resolver is the resolver used (the system resolver of the host).
	Resolver string `json:"resolver"`

	// Failure is the failure that occurred, or nil.
	Failure *string `json:"failure"`
}

func (m *Measurer) Run(
	ctx context.Context,
	args *model.ExperimentArgs) error {

	log.SetLevel(log.DebugLevel)
	measurement := args.Measurement

	tk := &TestKeys{}
	measurement.TestKeys = tk

	timeout := 60 * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	systemResolver := getSystemResolverAddress()
	if systemResolver == "" {
		return errors.New("could not get system resolver")
	}
	log.Infof("Using system resolver: %s", systemResolver)
	tk.Resolver = systemResolver

	// DDR queries are queries of the SVCB type for the _dns.resolver.arpa. domain.

	netx := &netxlite.Netx{}
	dialer := netx.NewDialerWithoutResolver(log.Log)
	transport := netxlite.NewUnwrappedDNSOverUDPTransport(
		dialer, systemResolver)
	encoder := &netxlite.DNSEncoderMiekg{}
	query := encoder.Encode(
		"_dns.resolver.arpa.", // As specified in RFC 9462
		dns.TypeSVCB,
		true)
	resp, err := transport.RoundTrip(ctx, query)
	if err != nil {
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
	} else {
		tk.DDRResponse = ddrResponse
	}

	tk.SupportsDDR = len(tk.DDRResponse) > 0

	log.Infof("Gathered DDR Responses: %+v", tk.DDRResponse)
	return nil
}

// decodeResponse decodes the response from the DNS query.
// DDR is only concerned with SVCB records, so we only decode those.
func decodeResponse(responseFields []dns.RR) ([]DDRResponse, error) {
	responses := make([]DDRResponse, 0)
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

func parseSvcb(rr *dns.SVCB) (DDRResponse, error) {
	keys := make(map[string]string)
	for _, kv := range rr.Value {
		value := kv.String()
		key := kv.Key().String()
		keys[key] = value
	}

	return DDRResponse{
		Priority: int(rr.Priority),
		Target:   rr.Target,
		Keys:     keys,
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

func NewExperimentMeasurer(config Config) model.ExperimentMeasurer {
	return &Measurer{
		config: config,
	}
}
