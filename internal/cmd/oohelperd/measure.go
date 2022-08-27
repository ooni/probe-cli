package main

//
// Top-level measurement algorithm
//

import (
	"context"
	"net"
	"net/url"
	"sync"

	"github.com/ooni/probe-cli/v3/internal/engine/experiment/webconnectivity"
)

type (
	// ctrlRequest is the request sent to the test helper
	ctrlRequest = webconnectivity.ControlRequest

	// ctrlResponse is the response from the test helper
	ctrlResponse = webconnectivity.ControlResponse
)

// measure performs the measurement described by the request and
// returns the corresponding response or an error.
func measure(ctx context.Context, config *handler, creq *ctrlRequest) (*ctrlResponse, error) {
	// parse input for correctness
	URL, err := url.Parse(creq.HTTPRequest)
	if err != nil {
		return nil, err
	}
	wg := &sync.WaitGroup{}

	// dns: start
	dnsch := make(chan ctrlDNSResult, 1)
	if net.ParseIP(URL.Hostname()) == nil {
		wg.Add(1)
		go dnsDo(ctx, &dnsConfig{
			Domain:      URL.Hostname(),
			NewResolver: config.NewResolver,
			Out:         dnsch,
			Wg:          wg,
		})
	}

	// wait for DNS measurements to complete
	wg.Wait()

	// start assembling the response
	cresp := new(ctrlResponse)
	select {
	case cresp.DNS = <-dnsch:
	default:
		// we need to emit a non-nil Addrs to match exactly
		// the behavior of the legacy TH
		cresp.DNS = ctrlDNSResult{
			Failure: nil,
			Addrs:   []string{},
		}
	}

	// figure out all the endpoints to measure
	endpoints := computeEndpoints(URL, creq, cresp.DNS.Addrs)

	// tcpconnect: start over all the endpoints
	tcpconnch := make(chan *tcpResultPair, len(endpoints))
	for _, endpoint := range endpoints {
		wg.Add(1)
		go tcpDo(ctx, &tcpConfig{
			EnableTLS:        endpoint.tls,
			Endpoint:         endpoint.epnt,
			NewDialer:        config.NewDialer,
			NewTSLHandshaker: config.NewTLSHandshaker,
			Out:              tcpconnch,
			URLHostname:      URL.Hostname(),
			Wg:               wg,
		})
	}

	// http: start
	httpch := make(chan ctrlHTTPResponse, 1)
	wg.Add(1)
	go httpDo(ctx, &httpConfig{
		Headers:           creq.HTTPRequestHeaders,
		MaxAcceptableBody: config.MaxAcceptableBody,
		NewClient:         config.NewClient,
		Out:               httpch,
		URL:               creq.HTTPRequest,
		Wg:                wg,
	})

	// wait for endpoint measurements to complete
	wg.Wait()

	// continue assembling the response
	cresp.HTTPRequest = <-httpch
	cresp.TCPConnect = make(map[string]ctrlTCPResult)
Loop:
	for {
		select {
		case tcpconn := <-tcpconnch:
			cresp.TCPConnect[tcpconn.Endpoint] = tcpconn.TCP
			if tcpconn.TLS != nil {
				cresp.TLSHandshake[tcpconn.Endpoint] = *tcpconn.TLS
			}
		default:
			break Loop
		}
	}

	return cresp, nil
}

// endpointInfo contains info about an endpoint to measure
type endpointInfo struct {
	epnt string
	tls  bool
}

// Computes all the endpoints that we need to measure including both the
// endpoints discovered by the probe and the ones discovered by us
func computeEndpoints(URL *url.URL, creq *ctrlRequest, addrs []string) (out []endpointInfo) {
	ports := []string{"80", "443"}
	if URL.Port() != "" {
		ports = []string{URL.Port()} // when there's a custom port just use that
	}
	mapping := make(map[string]int)
	for _, epnt := range creq.TCPConnect {
		addr, _, err := net.SplitHostPort(epnt)
		if err != nil {
			continue
		}
		mapping[addr]++
	}
	for _, addr := range addrs {
		if net.ParseIP(addr) != nil {
			mapping[addr]++
		}
	}
	for addr := range mapping {
		for _, port := range ports {
			epnt := net.JoinHostPort(addr, port)
			out = append(out, endpointInfo{
				epnt: epnt,
				tls:  port == "443",
			})
		}
	}
	return
}
