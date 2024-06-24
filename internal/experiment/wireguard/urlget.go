package wireguard

import (
	"io"
	"net/http"
	"time"

	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/model"
)

const (
	defaultURLGetTarget = "https://info.cern.ch/"
)

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

func (m *Measurer) urlget(url string, zeroTime time.Time, logger model.Logger) *URLGetResult {
	client := http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			DialContext:         m.tnet.DialContext,
			TLSHandshakeTimeout: 30 * time.Second,
		}}

	start := time.Since(zeroTime).Seconds()
	r, err := client.Get(url)
	if err != nil {
		logger.Warnf("urlget error: %v", err.Error())
		return newURLResultFromError(url, zeroTime, start, err)
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Warnf("urlget error: %v", err.Error())
		return newURLResultFromError(url, zeroTime, start, err)
	}
	defer r.Body.Close()

	return newURLResultWithStatusCode(url, zeroTime, start, r.StatusCode, body)
}
