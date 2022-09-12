package main

//
// Top-level measurement algorithm
//

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"sync"

	"github.com/ooni/probe-cli/v3/internal/model"
)

type (
	// ctrlRequest is the request sent to the test helper
	ctrlRequest = model.THRequest

	// ctrlResponse is the response from the test helper
	ctrlResponse = model.THResponse
)

// measure performs the measurement described by the request and
// returns the corresponding response or an error.
func measure(ctx context.Context, config *handler, creq *ctrlRequest) (*ctrlResponse, error) {
	// create indexed logger
	logger := &indexLogger{
		indexstr: fmt.Sprintf("<#%d> ", config.Indexer.Add(1)),
		logger:   config.BaseLogger,
	}

	// parse input for correctness
	URL, err := url.Parse(creq.HTTPRequest)
	if err != nil {
		logger.Warnf("cannot parse URL: %s", err.Error())
		return nil, err
	}
	wg := &sync.WaitGroup{}

	// dns: start
	dnsch := make(chan ctrlDNSResult, 1)
	if net.ParseIP(URL.Hostname()) == nil {
		wg.Add(1)
		go dnsDo(ctx, &dnsConfig{
			Cache:       config.DNSCache,
			Domain:      URL.Hostname(),
			Logger:      logger,
			NewResolver: config.NewResolver,
			Out:         dnsch,
			Wg:          wg,
		})
	}

	// wait for DNS measurements to complete
	wg.Wait()

	// start assembling the response
	cresp := &ctrlResponse{
		TCPConnect:   map[string]model.THTCPConnectResult{},
		TLSHandshake: map[string]model.THTLSHandshakeResult{},
		HTTPRequest:  model.THHTTPRequestResult{},
		DNS:          model.THDNSResult{},
		IPInfo:       map[string]*model.THIPInfo{},
	}
	select {
	case cresp.DNS = <-dnsch:
	default:
		// we need to emit a non-nil Addrs to match exactly
		// the behavior of the legacy TH
		cresp.DNS = ctrlDNSResult{
			Failure: nil,
			Addrs:   []string{},
			ASNs:    []int64{}, // unused by the TH and not serialized
		}
	}

	// obtain IP info and figure out the endpoints measurement plan
	cresp.IPInfo = newIPInfo(creq, cresp.DNS.Addrs)
	endpoints := ipInfoToEndpoints(URL, cresp.IPInfo)

	// tcpconnect: start over all the endpoints
	tcpconnch := make(chan *tcpResultPair, len(endpoints))
	for _, endpoint := range endpoints {
		wg.Add(1)
		go tcpDo(ctx, &tcpConfig{
			Address:          endpoint.Addr,
			Cache:            config.TCPCache,
			EnableTLS:        endpoint.TLS,
			Endpoint:         endpoint.Epnt,
			Logger:           logger,
			NewDialer:        config.NewDialer,
			NewTSLHandshaker: config.NewTLSHandshaker,
			URLHostname:      URL.Hostname(),
			Out:              tcpconnch,
			Wg:               wg,
		})
	}

	// http: start
	httpch := make(chan ctrlHTTPResponse, 1)
	wg.Add(1)
	go httpDo(ctx, &httpConfig{
		Cache:             config.HTTPCache,
		Headers:           creq.HTTPRequestHeaders,
		Logger:            logger,
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
Loop:
	for {
		select {
		case tcpconn := <-tcpconnch:
			cresp.TCPConnect[tcpconn.Endpoint] = tcpconn.TCP
			if tcpconn.TLS != nil {
				cresp.TLSHandshake[tcpconn.Endpoint] = *tcpconn.TLS
				if info := cresp.IPInfo[tcpconn.Address]; info != nil && tcpconn.TLS.Failure == nil {
					info.Flags |= model.THIPInfoFlagValidForDomain
				}
			}
		default:
			break Loop
		}
	}

	return cresp, nil
}
