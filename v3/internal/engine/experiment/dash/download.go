package dash

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type downloadDeps interface {
	HTTPClient() *http.Client
	NewHTTPRequest(method string, url string, body io.Reader) (*http.Request, error)
	ReadAll(r io.Reader) ([]byte, error)
	Scheme() string
	UserAgent() string
}

type downloadConfig struct {
	authorization string
	begin         time.Time
	currentRate   int64
	deps          downloadDeps
	elapsedTarget int64
	fqdn          string
}

type downloadResult struct {
	elapsed      float64
	received     int64
	requestTicks float64
	serverURL    string
	timestamp    int64
}

func download(ctx context.Context, config downloadConfig) (downloadResult, error) {
	nbytes := (config.currentRate * 1000 * config.elapsedTarget) >> 3
	var URL url.URL
	URL.Scheme = config.deps.Scheme()
	URL.Host = config.fqdn
	URL.Path = fmt.Sprintf("%s%d", downloadPath, nbytes)
	req, err := config.deps.NewHTTPRequest("GET", URL.String(), nil)
	var result downloadResult
	if err != nil {
		return result, err
	}
	result.serverURL = URL.String()
	req.Header.Set("User-Agent", config.deps.UserAgent())
	req.Header.Set("Authorization", config.authorization)
	savedTicks := time.Now()
	resp, err := config.deps.HTTPClient().Do(req.WithContext(ctx))
	if err != nil {
		return result, err
	}
	if resp.StatusCode != 200 {
		return result, errHTTPRequestFailed
	}
	defer resp.Body.Close()
	data, err := config.deps.ReadAll(resp.Body)
	if err != nil {
		return result, err
	}
	// Implementation note: MK contains a comment that says that Neubot uses
	// the elapsed time since when we start receiving the response but it
	// turns out that Neubot and MK do the same. So, we do what they do. At
	// the same time, we are currently not able to include the overhead that
	// is caused by HTTP headers etc. So, we're a bit less precise.
	result.elapsed = time.Now().Sub(savedTicks).Seconds()
	result.received = int64(len(data))
	result.requestTicks = savedTicks.Sub(config.begin).Seconds()
	result.timestamp = time.Now().Unix()
	return result, nil
}
