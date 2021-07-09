package internal

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"sync"

	"github.com/ooni/probe-cli/v3/internal/engine/experiment/webconnectivity"
	"github.com/ooni/probe-cli/v3/internal/engine/netx"
)

type (
	// CtrlRequest is the request sent to the test helper
	CtrlRequest = webconnectivity.ControlRequest

	// CtrlResponse is the response from the test helper
	CtrlResponse = webconnectivity.ControlResponse
)

// MeasureConfig contains configuration for Measure.
type MeasureConfig struct {
	Client            *http.Client
	Dialer            netx.Dialer
	MaxAcceptableBody int64
	Resolver          netx.Resolver
}

// Measure performs the measurement described by the request and
// returns the corresponding response or an error.
func Measure(ctx context.Context, config MeasureConfig, creq *CtrlRequest) (*CtrlResponse, error) {
	// Regexp taken from: https://github.com/citizenlab/test-lists/blob/master/scripts/lint-lists.py#L18
	urlRegexp := regexp.MustCompile(`^(?:http)s?://(?:(?:[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)+(?:[a-zA-Z]{2,6}\.?|[a-zA-Z0-9-]{2,}\.?)|\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3})(?::\d+)?(?:/?|[/?]\S+)$`)
	if urlRegexp.Match([]byte(creq.HTTPRequest)) == false {
		return nil, errors.New("invalid URL")
	}
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
		// we land here when there's no domain name
	}
	cresp.HTTPRequest = <-httpch
	cresp.TCPConnect = make(map[string]CtrlTCPResult)
	for len(cresp.TCPConnect) < len(creq.TCPConnect) {
		tcpconn := <-tcpconnch
		cresp.TCPConnect[tcpconn.Endpoint] = tcpconn.Result
	}
	return cresp, nil
}
