// Package fbmessenger contains the Facebook Messenger network experiment.
//
// See https://github.com/ooni/spec/blob/master/nettests/ts-019-facebook-messenger.md
package fbmessenger

import (
	"context"
	"errors"
	"math/rand"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine/experiment/urlgetter"
	"github.com/ooni/probe-cli/v3/internal/engine/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite/errorsx"
)

const (
	// FacebookASN is Facebook's ASN
	FacebookASN = 32934

	// ServiceSTUN is the STUN service
	ServiceSTUN = "dnslookup://stun.fbsbx.com"

	// ServiceBAPI is the b-api service
	ServiceBAPI = "tcpconnect://b-api.facebook.com:443"

	// ServiceBGraph is the b-graph service
	ServiceBGraph = "tcpconnect://b-graph.facebook.com:443"

	// ServiceEdge is the edge service
	ServiceEdge = "tcpconnect://edge-mqtt.facebook.com:443"

	// ServiceExternalCDN is the external CDN service
	ServiceExternalCDN = "tcpconnect://external.xx.fbcdn.net:443"

	// ServiceScontentCDN is the scontent CDN service
	ServiceScontentCDN = "tcpconnect://scontent.xx.fbcdn.net:443"

	// ServiceStar is the star service
	ServiceStar = "tcpconnect://star.c10r.facebook.com:443"

	testName    = "facebook_messenger"
	testVersion = "0.2.0"
)

// Config contains the experiment config.
type Config struct{}

// TestKeys contains the experiment results
type TestKeys struct {
	urlgetter.TestKeys
	FacebookBAPIDNSConsistent        *bool `json:"facebook_b_api_dns_consistent"`
	FacebookBAPIReachable            *bool `json:"facebook_b_api_reachable"`
	FacebookBGraphDNSConsistent      *bool `json:"facebook_b_graph_dns_consistent"`
	FacebookBGraphReachable          *bool `json:"facebook_b_graph_reachable"`
	FacebookEdgeDNSConsistent        *bool `json:"facebook_edge_dns_consistent"`
	FacebookEdgeReachable            *bool `json:"facebook_edge_reachable"`
	FacebookExternalCDNDNSConsistent *bool `json:"facebook_external_cdn_dns_consistent"`
	FacebookExternalCDNReachable     *bool `json:"facebook_external_cdn_reachable"`
	FacebookScontentCDNDNSConsistent *bool `json:"facebook_scontent_cdn_dns_consistent"`
	FacebookScontentCDNReachable     *bool `json:"facebook_scontent_cdn_reachable"`
	FacebookStarDNSConsistent        *bool `json:"facebook_star_dns_consistent"`
	FacebookStarReachable            *bool `json:"facebook_star_reachable"`
	FacebookSTUNDNSConsistent        *bool `json:"facebook_stun_dns_consistent"`
	FacebookSTUNReachable            *bool `json:"facebook_stun_reachable"`
	FacebookDNSBlocking              *bool `json:"facebook_dns_blocking"`
	FacebookTCPBlocking              *bool `json:"facebook_tcp_blocking"`
}

// Update updates the TestKeys using the given MultiOutput result.
func (tk *TestKeys) Update(v urlgetter.MultiOutput) {
	// Update the easy to update entries first
	tk.NetworkEvents = append(tk.NetworkEvents, v.TestKeys.NetworkEvents...)
	tk.Queries = append(tk.Queries, v.TestKeys.Queries...)
	tk.Requests = append(tk.Requests, v.TestKeys.Requests...)
	tk.TCPConnect = append(tk.TCPConnect, v.TestKeys.TCPConnect...)
	tk.TLSHandshakes = append(tk.TLSHandshakes, v.TestKeys.TLSHandshakes...)
	// Set the status of endpoints
	switch v.Input.Target {
	case ServiceSTUN:
		var ignored *bool
		tk.ComputeEndpointStatus(v, &tk.FacebookSTUNDNSConsistent, &ignored)
	case ServiceBAPI:
		tk.ComputeEndpointStatus(
			v, &tk.FacebookBAPIDNSConsistent, &tk.FacebookBAPIReachable)
	case ServiceBGraph:
		tk.ComputeEndpointStatus(
			v, &tk.FacebookBGraphDNSConsistent, &tk.FacebookBGraphReachable)
	case ServiceEdge:
		tk.ComputeEndpointStatus(
			v, &tk.FacebookEdgeDNSConsistent, &tk.FacebookEdgeReachable)
	case ServiceExternalCDN:
		tk.ComputeEndpointStatus(
			v, &tk.FacebookExternalCDNDNSConsistent, &tk.FacebookExternalCDNReachable)
	case ServiceScontentCDN:
		tk.ComputeEndpointStatus(
			v, &tk.FacebookScontentCDNDNSConsistent, &tk.FacebookScontentCDNReachable)
	case ServiceStar:
		tk.ComputeEndpointStatus(
			v, &tk.FacebookStarDNSConsistent, &tk.FacebookStarReachable)
	}
}

var (
	trueValue  = true
	falseValue = false
)

// ComputeEndpointStatus computes the DNS and TCP status of a specific endpoint.
func (tk *TestKeys) ComputeEndpointStatus(v urlgetter.MultiOutput, dns, tcp **bool) {
	// start where all is unknown
	*dns, *tcp = nil, nil
	// process DNS first
	if v.TestKeys.FailedOperation != nil && *v.TestKeys.FailedOperation == errorsx.ResolveOperation {
		tk.FacebookDNSBlocking = &trueValue
		*dns = &falseValue
		return // we know that the DNS has failed
	}
	for _, query := range v.TestKeys.Queries {
		for _, ans := range query.Answers {
			if ans.ASN != FacebookASN {
				tk.FacebookDNSBlocking = &trueValue
				*dns = &falseValue
				return // because DNS is lying
			}
		}
	}
	*dns = &trueValue
	// now process connect
	if v.TestKeys.FailedOperation != nil && *v.TestKeys.FailedOperation == errorsx.ConnectOperation {
		tk.FacebookTCPBlocking = &trueValue
		*tcp = &falseValue
		return // because connect failed
	}
	// all good
	*tcp = &trueValue
}

// Measurer performs the measurement
type Measurer struct {
	// Config contains the experiment settings. If empty we
	// will be using default settings.
	Config Config

	// Getter is an optional getter to be used for testing.
	Getter urlgetter.MultiGetter
}

// ExperimentName implements ExperimentMeasurer.ExperimentName
func (m Measurer) ExperimentName() string {
	return testName
}

// ExperimentVersion implements ExperimentMeasurer.ExperimentVersion
func (m Measurer) ExperimentVersion() string {
	return testVersion
}

// Run implements ExperimentMeasurer.Run
func (m Measurer) Run(
	ctx context.Context, sess model.ExperimentSession,
	measurement *model.Measurement, callbacks model.ExperimentCallbacks,
) error {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	urlgetter.RegisterExtensions(measurement)
	// generate targets
	services := []string{
		ServiceSTUN, ServiceBAPI, ServiceBGraph, ServiceEdge, ServiceExternalCDN,
		ServiceScontentCDN, ServiceStar,
	}
	var inputs []urlgetter.MultiInput
	for _, service := range services {
		inputs = append(inputs, urlgetter.MultiInput{Target: service})
	}
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	rnd.Shuffle(len(inputs), func(i, j int) {
		inputs[i], inputs[j] = inputs[j], inputs[i]
	})
	// measure in parallel
	multi := urlgetter.Multi{Begin: time.Now(), Getter: m.Getter, Session: sess}
	testkeys := new(TestKeys)
	testkeys.Agent = "redirect"
	measurement.TestKeys = testkeys
	for entry := range multi.Collect(ctx, inputs, "facebook_messenger", callbacks) {
		testkeys.Update(entry)
	}
	// if we haven't yet determined the status of DNS blocking and TCP blocking
	// then no blocking has been detected and we can set them
	if testkeys.FacebookDNSBlocking == nil {
		testkeys.FacebookDNSBlocking = &falseValue
	}
	if testkeys.FacebookTCPBlocking == nil {
		testkeys.FacebookTCPBlocking = &falseValue
	}
	return nil
}

// NewExperimentMeasurer creates a new ExperimentMeasurer.
func NewExperimentMeasurer(config Config) model.ExperimentMeasurer {
	return Measurer{Config: config}
}

// SummaryKeys contains summary keys for this experiment.
//
// Note that this structure is part of the ABI contract with probe-cli
// therefore we should be careful when changing it.
type SummaryKeys struct {
	DNSBlocking bool `json:"facebook_dns_blocking"`
	TCPBlocking bool `json:"facebook_tcp_blocking"`
	IsAnomaly   bool `json:"-"`
}

// GetSummaryKeys implements model.ExperimentMeasurer.GetSummaryKeys.
func (m Measurer) GetSummaryKeys(measurement *model.Measurement) (interface{}, error) {
	sk := SummaryKeys{IsAnomaly: false}
	tk, ok := measurement.TestKeys.(*TestKeys)
	if !ok {
		return sk, errors.New("invalid test keys type")
	}
	dnsBlocking := tk.FacebookDNSBlocking != nil && *tk.FacebookDNSBlocking
	tcpBlocking := tk.FacebookTCPBlocking != nil && *tk.FacebookTCPBlocking
	sk.DNSBlocking = dnsBlocking
	sk.TCPBlocking = tcpBlocking
	sk.IsAnomaly = dnsBlocking || tcpBlocking
	return sk, nil
}
