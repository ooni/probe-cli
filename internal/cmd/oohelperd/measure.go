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

	// tcpconnect: start
	tcpconnch := make(chan tcpResultPair, len(creq.TCPConnect))
	for _, endpoint := range creq.TCPConnect {
		wg.Add(1)
		go tcpDo(ctx, &tcpConfig{
			Endpoint:  endpoint,
			NewDialer: config.NewDialer,
			Out:       tcpconnch,
			Wg:        wg,
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
			cresp.TCPConnect[tcpconn.Endpoint] = tcpconn.Result
		default:
			break Loop
		}
	}

	return cresp, nil
}
