package wireguard

import (
	"context"
	"net/http"
	"time"

	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

const (
	// defaultURLGetTarget is the web page that the experiment will fetch by default.
	defaultURLGetTarget = "https://info.cern.ch/"
)

// urlget implements an straightforward urlget experiment using the standard library.
// By default we pass the wireguard tunnel DialContext to the `http.Transport` on the `http.Client` creation.
func (m *Measurer) urlget(ctx context.Context, url string, zeroTime time.Time, logger model.Logger) *URLGetResult {
	if m.dialContextFn == nil {
		m.dialContextFn = m.tnet.DialContext
	}
	if m.httpClient == nil {
		m.httpClient = &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				DialContext:         m.dialContextFn,
				TLSHandshakeTimeout: 30 * time.Second,
			}}
	}

	start := time.Since(zeroTime).Seconds()
	r, err := m.httpClient.Get(url)
	if err != nil {
		logger.Warnf("urlget error: %v", err.Error())
		return newURLResultFromError(url, zeroTime, start, err)
	}
	body, err := netxlite.ReadAllContext(ctx, r.Body)
	if err != nil {
		logger.Warnf("urlget error: %v", err.Error())
		return newURLResultFromError(url, zeroTime, start, err)
	}
	defer r.Body.Close()

	return newURLResultWithStatusCode(url, zeroTime, start, r.StatusCode, body)
}

func newURLResultFromError(url string, zeroTime time.Time, start float64, err error) *URLGetResult {
	return &URLGetResult{
		URL:     url,
		T0:      start,
		T:       time.Since(zeroTime).Seconds(),
		Failure: measurexlite.NewFailure(err),
		Error:   err.Error(),
	}
}

func newURLResultWithStatusCode(url string, zeroTime time.Time, start float64, statusCode int, body []byte) *URLGetResult {
	return &URLGetResult{
		ByteCount:  len(body),
		URL:        url,
		T0:         start,
		T:          time.Since(zeroTime).Seconds(),
		StatusCode: statusCode,
	}
}
