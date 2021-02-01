package dash

import (
	"io"
	"net/http"
	"time"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/engine/model"
)

type FakeDeps struct {
	httpTransport        http.RoundTripper
	jsonMarshalErr       error
	jsonMarshalResult    []byte
	newHTTPRequestErr    error
	newHTTPRequestResult *http.Request
	readAllErr           error
	readAllResult        []byte
}

func (d FakeDeps) HTTPClient() *http.Client {
	return &http.Client{Transport: d.httpTransport}
}

func (d FakeDeps) JSONMarshal(v interface{}) ([]byte, error) {
	return d.jsonMarshalResult, d.jsonMarshalErr
}

func (d FakeDeps) Logger() model.Logger {
	return log.Log
}

func (d FakeDeps) NewHTTPRequest(
	method string, url string, body io.Reader) (*http.Request, error) {
	return d.newHTTPRequestResult, d.newHTTPRequestErr
}

func (d FakeDeps) ReadAll(r io.Reader) ([]byte, error) {
	return d.readAllResult, d.readAllErr
}

func (d FakeDeps) Scheme() string {
	return "https"
}

func (d FakeDeps) UserAgent() string {
	return "miniooni/0.1.0-dev"
}

type FakeHTTPTransport struct {
	err  error
	resp *http.Response
}

func (txp FakeHTTPTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	time.Sleep(10 * time.Microsecond)
	return txp.resp, txp.err
}

type FakeHTTPTransportStack struct {
	all []FakeHTTPTransport
}

func (txp *FakeHTTPTransportStack) RoundTrip(req *http.Request) (*http.Response, error) {
	frame := txp.all[0]
	txp.all = txp.all[1:]
	return frame.RoundTrip(req)
}
