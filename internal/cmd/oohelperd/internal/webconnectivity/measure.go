package webconnectivity

import (
	"context"
	"net"
	"net/url"
	"sync"

	"github.com/ooni/probe-cli/v3/internal/engine/experiment/webconnectivity"
	"github.com/ooni/probe-cli/v3/internal/model"
)

type (
	// CtrlRequest is the request sent to the test helper
	CtrlRequest = webconnectivity.ControlRequest

	// CtrlResponse is the response from the test helper
	CtrlResponse = webconnectivity.ControlResponse
)

// MeasureConfig contains configuration for Measure.
type MeasureConfig struct {
	Client            model.HTTPClient
	Dialer            model.Dialer
	MaxAcceptableBody int64
	Resolver          model.Resolver
}

// Measure performs the measurement described by the request and
// returns the corresponding response or an error.
func Measure(ctx context.Context, config MeasureConfig, creq *CtrlRequest) (*CtrlResponse, error) {
	// parse input for correctness
	URL, err := url.Parse(creq.HTTPRequest)
	if err != nil {
		return nil, err
	}
	// dns: start
	wg := new(sync.WaitGroup)
	dnsch := make(chan CtrlDNSResult, 1)
	if net.ParseIP(URL.Hostname()) == nil {
		wg.Add(1)
		go DNSDo(ctx, &DNSConfig{
			Domain:   URL.Hostname(),
			Out:      dnsch,
			Resolver: config.Resolver,
			Wg:       wg,
		})
	}
	// tcpconnect: start
	tcpconnch := make(chan TCPResultPair, len(creq.TCPConnect))
	for _, endpoint := range creq.TCPConnect {
		wg.Add(1)
		go TCPDo(ctx, &TCPConfig{
			Dialer:   config.Dialer,
			Endpoint: endpoint,
			Out:      tcpconnch,
			Wg:       wg,
		})
	}
	// http: start
	httpch := make(chan CtrlHTTPResponse, 1)
	wg.Add(1)
	go HTTPDo(ctx, &HTTPConfig{
		Client:            config.Client,
		Headers:           creq.HTTPRequestHeaders,
		MaxAcceptableBody: config.MaxAcceptableBody,
		Out:               httpch,
		URL:               creq.HTTPRequest,
		Wg:                wg,
	})
	// wait for measurement steps to complete
	wg.Wait()
	// assemble response
	cresp := new(CtrlResponse)
	select {
	case cresp.DNS = <-dnsch:
	default:
		// we need to emit a non-nil Addrs to match exactly
		// the behavior of the legacy TH
		cresp.DNS = CtrlDNSResult{
			Failure: nil,
			Addrs:   []string{},
		}
	}
	cresp.HTTPRequest = <-httpch
	cresp.TCPConnect = make(map[string]CtrlTCPResult)
	for len(cresp.TCPConnect) < len(creq.TCPConnect) {
		tcpconn := <-tcpconnch
		cresp.TCPConnect[tcpconn.Endpoint] = tcpconn.Result
	}
	return cresp, nil
}
