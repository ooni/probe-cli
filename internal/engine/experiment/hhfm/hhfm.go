// Package hhfm contains the HTTP Header Field Manipulation network experiment.
//
// See https://github.com/ooni/spec/blob/master/nettests/ts-006-header-field-manipulation.md
package hhfm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sort"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine/experiment/urlgetter"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/randx"
	"github.com/ooni/probe-cli/v3/internal/tracex"
)

const (
	testName    = "http_header_field_manipulation"
	testVersion = "0.2.0"
)

// Config contains the experiment config.
type Config struct{}

// TestKeys contains the experiment test keys.
//
// Here we are emitting for the same set of test keys that are
// produced by the MK implementation.
type TestKeys struct {
	Agent      string                `json:"agent"`
	Failure    *string               `json:"failure"`
	Requests   []tracex.RequestEntry `json:"requests"`
	SOCKSProxy *string               `json:"socksproxy"`
	Tampering  Tampering             `json:"tampering"`
}

// Tampering describes the detected forms of tampering.
//
// The meaning of these fields is described in the specification.
type Tampering struct {
	HeaderFieldName           bool     `json:"header_field_name"`
	HeaderFieldNumber         bool     `json:"header_field_number"`
	HeaderFieldValue          bool     `json:"header_field_value"`
	HeaderNameCapitalization  bool     `json:"header_name_capitalization"`
	HeaderNameDiff            []string `json:"header_name_diff"`
	RequestLineCapitalization bool     `json:"request_line_capitalization"`
	Total                     bool     `json:"total"`
}

// NewExperimentMeasurer creates a new ExperimentMeasurer.
func NewExperimentMeasurer(config Config) model.ExperimentMeasurer {
	return Measurer{Config: config}
}

// Transport is the definition of http.RoundTripper used by this package.
type Transport interface {
	RoundTrip(req *http.Request) (*http.Response, error)
	CloseIdleConnections()
}

// Measurer performs the measurement.
type Measurer struct {
	Config    Config
	Transport Transport // for testing
}

// ExperimentName implements ExperimentMeasurer.ExperiExperimentName.
func (m Measurer) ExperimentName() string {
	return testName
}

// ExperimentVersion implements ExperimentMeasurer.ExperimentVersion.
func (m Measurer) ExperimentVersion() string {
	return testVersion
}

var (
	// ErrNoAvailableTestHelpers is emitted when there are no available test helpers.
	ErrNoAvailableTestHelpers = errors.New("no available helpers")

	// ErrInvalidHelperType is emitted when the helper type is invalid.
	ErrInvalidHelperType = errors.New("invalid helper type")
)

// Run implements ExperimentMeasurer.Run.
func (m Measurer) Run(ctx context.Context, args *model.ExperimentArgs) error {
	callbacks := args.Callbacks
	measurement := args.Measurement
	sess := args.Session
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	urlgetter.RegisterExtensions(measurement)
	tk := new(TestKeys)
	tk.Agent = "agent"
	tk.Tampering.HeaderNameDiff = []string{}
	measurement.TestKeys = tk
	// parse helper
	const helperName = "http-return-json-headers"
	helpers, ok := sess.GetTestHelpersByName(helperName)
	if !ok || len(helpers) < 1 {
		return ErrNoAvailableTestHelpers
	}
	helper := helpers[0]
	if helper.Type != "legacy" {
		return ErrInvalidHelperType
	}
	measurement.TestHelpers = map[string]interface{}{
		"backend": helper.Address,
	}
	// prepare request
	req, err := http.NewRequest("GeT", helper.Address, nil)
	if err != nil {
		return err
	}
	headers := map[string]string{
		randx.ChangeCapitalization("Accept"):          model.HTTPHeaderAccept,
		randx.ChangeCapitalization("Accept-Charset"):  "ISO-8859-1,utf-8;q=0.7,*;q=0.3",
		randx.ChangeCapitalization("Accept-Encoding"): "gzip,deflate,sdch",
		randx.ChangeCapitalization("Accept-Language"): model.HTTPHeaderAcceptLanguage,
		randx.ChangeCapitalization("Host"):            randx.Letters(15) + ".com",
		randx.ChangeCapitalization("User-Agent"):      model.HTTPHeaderUserAgent,
	}
	for key, value := range headers {
		// Implementation note: Golang will normalize the header names. We will use
		// a custom dialer to restore the random capitalisation.
		req.Header.Set(key, value)
	}
	req.Host = req.Header.Get("Host")
	// fill tk.Requests[0]
	tk.Requests = NewRequestEntryList(req, headers)
	// prepare transport
	txp := m.Transport
	if txp == nil {
		ht := http.DefaultTransport.(*http.Transport).Clone() // basically: use defaults
		ht.DisableCompression = true                          // disable sending Accept: gzip
		ht.ForceAttemptHTTP2 = false
		ht.DialContext = Dialer{Headers: headers}.DialContext
		txp = ht
	}
	defer txp.CloseIdleConnections()
	// round trip and read body
	// TODO(bassosimone): this implementation will lead to false positives if the
	// network is really bad. Yet, this seems what MK does, so I'd rather start
	// from that and then see to improve the robustness in the future.
	resp, data, err := Transact(txp, req.WithContext(ctx), callbacks)
	if err != nil {
		tk.Failure = tracex.NewFailure(err)
		tk.Requests[0].Failure = tk.Failure
		tk.Tampering.Total = true
		return nil // measurement did not fail, we measured tampering
	}
	// fill tk.Requests[0].Response
	tk.Requests[0].Response = NewHTTPResponse(resp, data)
	// parse response body
	var jsonHeaders JSONHeaders
	if err := json.Unmarshal(data, &jsonHeaders); err != nil {
		failure := netxlite.FailureJSONParseError
		tk.Failure = &failure
		tk.Tampering.Total = true
		return nil // measurement did not fail, we measured tampering
	}
	// fill tampering
	tk.FillTampering(req, jsonHeaders, headers)
	return nil
}

// Transact performs the HTTP transaction which consists of performing
// the HTTP round trip and then reading the body.
func Transact(txp Transport, req *http.Request,
	callbacks model.ExperimentCallbacks) (*http.Response, []byte, error) {
	// make sure that we return a wrapped error here
	resp, data, err := transact(txp, req, callbacks)
	if err != nil {
		err = netxlite.NewTopLevelGenericErrWrapper(err)
	}
	return resp, data, err
}

func transact(txp Transport, req *http.Request,
	callbacks model.ExperimentCallbacks) (*http.Response, []byte, error) {
	callbacks.OnProgress(0.25, "sending request...")
	resp, err := txp.RoundTrip(req)
	callbacks.OnProgress(0.50, fmt.Sprintf("got reseponse headers... %+v", err))
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, nil, urlgetter.ErrHTTPRequestFailed
	}
	callbacks.OnProgress(0.75, "reading response body...")
	data, err := netxlite.ReadAllContext(req.Context(), resp.Body)
	callbacks.OnProgress(1.00, fmt.Sprintf("got reseponse body... %+v", err))
	if err != nil {
		return nil, nil, err
	}
	return resp, data, nil
}

// FillTampering fills the tampering structure in the TestKeys
// based on the value of other fields of the TestKeys, the original
// HTTP request, the response from the test helper, and the
// headers with modified capitalisation.
func (tk *TestKeys) FillTampering(
	req *http.Request, jsonHeaders JSONHeaders, headers map[string]string) {
	tk.Tampering.RequestLineCapitalization = (fmt.Sprintf(
		"%s / HTTP/1.1", req.Method) != jsonHeaders.RequestLine)
	tk.Tampering.HeaderFieldNumber = len(headers) != len(jsonHeaders.HeadersDict)
	expectedHeaderKeys := make(map[string]string)
	for key := range headers {
		expectedHeaderKeys[http.CanonicalHeaderKey(key)] = key
	}
	receivedHeaderKeys := make(map[string]string)
	for key := range jsonHeaders.HeadersDict {
		receivedHeaderKeys[http.CanonicalHeaderKey(key)] = key
	}
	commonHeaderKeys := make(map[string]int)
	for key := range expectedHeaderKeys {
		commonHeaderKeys[key]++
	}
	for key := range receivedHeaderKeys {
		commonHeaderKeys[key]++
	}
	for key, count := range commonHeaderKeys {
		if count != 2 {
			continue // not in common
		}
		expectedKey, receivedKey := expectedHeaderKeys[key], receivedHeaderKeys[key]
		if expectedKey != receivedKey {
			tk.Tampering.HeaderNameCapitalization = true
			tk.Tampering.HeaderNameDiff = append(tk.Tampering.HeaderNameDiff, expectedKey)
			tk.Tampering.HeaderNameDiff = append(tk.Tampering.HeaderNameDiff, receivedKey)
		}
		expectedValue := headers[expectedKey]
		receivedValue := jsonHeaders.HeadersDict[receivedKey]
		if len(receivedValue) != 1 || expectedValue != receivedValue[0] {
			tk.Tampering.HeaderFieldValue = true
		}
	}
}

// NewRequestEntryList creates a new []tracex.RequestEntry given a
// specific *http.Request and headers with random case.
func NewRequestEntryList(req *http.Request, headers map[string]string) (out []tracex.RequestEntry) {
	out = []tracex.RequestEntry{{
		Request: tracex.HTTPRequest{
			Headers:     make(map[string]tracex.MaybeBinaryValue),
			HeadersList: []tracex.HTTPHeader{},
			Method:      req.Method,
			URL:         req.URL.String(),
		},
	}}
	for key, value := range headers {
		// Using the random capitalization headers here
		mbv := tracex.MaybeBinaryValue{Value: value}
		out[0].Request.Headers[key] = mbv
		out[0].Request.HeadersList = append(out[0].Request.HeadersList,
			tracex.HTTPHeader{Key: key, Value: mbv})
	}
	sort.Slice(out[0].Request.HeadersList, func(i, j int) bool {
		return out[0].Request.HeadersList[i].Key < out[0].Request.HeadersList[j].Key
	})
	return
}

// NewHTTPResponse creates a new tracex.HTTPResponse given a
// specific *http.Response instance and its body.
func NewHTTPResponse(resp *http.Response, data []byte) (out tracex.HTTPResponse) {
	out = tracex.HTTPResponse{
		Body:        tracex.HTTPBody{Value: string(data)},
		Code:        int64(resp.StatusCode),
		Headers:     make(map[string]tracex.MaybeBinaryValue),
		HeadersList: []tracex.HTTPHeader{},
	}
	for key := range resp.Header {
		mbv := tracex.MaybeBinaryValue{Value: resp.Header.Get(key)}
		out.Headers[key] = mbv
		out.HeadersList = append(out.HeadersList, tracex.HTTPHeader{Key: key, Value: mbv})
	}
	sort.Slice(out.HeadersList, func(i, j int) bool {
		return out.HeadersList[i].Key < out.HeadersList[j].Key
	})
	return
}

// JSONHeaders contains the response from the backend server.
//
// Here we're defining only the fields we care about.
type JSONHeaders struct {
	HeadersDict map[string][]string `json:"headers_dict"`
	RequestLine string              `json:"request_line"`
}

// Dialer is a dialer that performs headers transformations.
//
// Because Golang will canonicalize header names, we need to reintroduce
// the random capitalization when emitting the request.
//
// This implementation rests on the assumption that we shall use the
// same connection just once, which is guarantee by the implementation
// of HHFM above. If using this code elsewhere, make sure that you
// guarantee that the connection is used for a single request and that
// such a request does not contain any body.
type Dialer struct {
	Dialer  model.SimpleDialer // used for testing
	Headers map[string]string
}

// DialContext dials a specific connection and arranges such that
// headers in the outgoing request are transformed.
func (d Dialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	child := d.Dialer
	if child == nil {
		// TODO(bassosimone): figure out why using dialer.New here
		// causes the experiment to fail with eof_error
		child = &net.Dialer{Timeout: 15 * time.Second}
	}
	conn, err := child.DialContext(ctx, network, address)
	if err != nil {
		return nil, err
	}
	return Conn{Conn: conn, Headers: d.Headers}, nil
}

// Conn is a connection where headers in the outgoing request
// are transformed according to a transform table.
type Conn struct {
	net.Conn
	Headers map[string]string
}

// Write implements Conn.Write.
func (c Conn) Write(b []byte) (int, error) {
	for key := range c.Headers {
		b = bytes.Replace(b, []byte(http.CanonicalHeaderKey(key)+":"), []byte(key+":"), 1)
	}
	return c.Conn.Write(b)
}

// SummaryKeys contains summary keys for this experiment.
//
// Note that this structure is part of the ABI contract with ooniprobe
// therefore we should be careful when changing it.
type SummaryKeys struct {
	IsAnomaly bool `json:"-"`
}

// GetSummaryKeys implements model.ExperimentMeasurer.GetSummaryKeys.
func (m Measurer) GetSummaryKeys(measurement *model.Measurement) (interface{}, error) {
	sk := SummaryKeys{IsAnomaly: false}
	tk, ok := measurement.TestKeys.(*TestKeys)
	if !ok {
		return sk, errors.New("invalid test keys type")
	}
	sk.IsAnomaly = (tk.Tampering.HeaderFieldName ||
		tk.Tampering.HeaderFieldNumber ||
		tk.Tampering.HeaderFieldValue ||
		tk.Tampering.HeaderNameCapitalization ||
		tk.Tampering.RequestLineCapitalization ||
		tk.Tampering.Total)
	return sk, nil
}
