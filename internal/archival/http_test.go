package archival

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

func TestSaverHTTPRoundTrip(t *testing.T) {
	// newHTTPTransport creates a new HTTP transport for testing.
	newHTTPTransport := func(resp *http.Response, err error) model.HTTPTransport {
		return &mocks.HTTPTransport{
			MockRoundTrip: func(req *http.Request) (*http.Response, error) {
				return resp, err
			},
			MockNetwork: func() string {
				return "tcp"
			},
		}
	}

	// successFlowWithBody is a successful test case with possible body truncation.
	successFlowWithBody := func(realBody []byte, maxBodySize int64) error {
		// truncate the expected body if required
		expectedBody := realBody
		truncated := false
		if int64(len(realBody)) > maxBodySize {
			expectedBody = realBody[:maxBodySize]
			truncated = true
		}
		// construct the saver and the validator
		saver := NewSaver()
		v := &SingleHTTPRoundTripValidator{
			ExpectFailure: nil,
			ExpectMethod:  "GET",
			ExpectRequestHeaders: map[string][]string{
				"Host":        {"127.0.0.1:8080"},
				"User-Agent":  {"antani/1.0"},
				"X-Client-IP": {"130.192.91.211"},
			},
			ExpectResponseBody:            expectedBody,
			ExpectResponseBodyIsTruncated: truncated,
			ExpectResponseBodyLength:      int64(len(expectedBody)),
			ExpectResponseHeaders: map[string][]string{
				"Server":       {"antani/1.0"},
				"Content-Type": {"text/plain"},
			},
			ExpectStatusCode: 200,
			ExpectTransport:  "tcp",
			ExpectURL:        "http://127.0.0.1:8080/antani",
			RealResponseBody: realBody,
			Saver:            saver,
		}
		// construct transport and perform the HTTP round trip
		txp := newHTTPTransport(v.NewHTTPResponse(), nil)
		resp, err := saver.HTTPRoundTrip(txp, maxBodySize, v.NewHTTPRequest())
		if err != nil {
			return err
		}
		if resp == nil {
			return errors.New("expected non-nil resp")
		}
		// ensure that we can still read the _full_ response body
		ctx := context.Background()
		data, err := netxlite.ReadAllContext(ctx, resp.Body)
		if err != nil {
			return err
		}
		if diff := cmp.Diff(realBody, data); diff != "" {
			return errors.New(diff)
		}
		// validate the content of the trace
		return v.Validate()
	}

	t.Run("on success without truncation", func(t *testing.T) {
		realBody := []byte("0xdeadbeef")
		const maxBodySize = 1 << 20
		err := successFlowWithBody(realBody, maxBodySize)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("on success with truncation", func(t *testing.T) {
		realBody := []byte("0xdeadbeef")
		const maxBodySize = 4
		err := successFlowWithBody(realBody, maxBodySize)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("on failure during round trip", func(t *testing.T) {
		expectedError := netxlite.NewTopLevelGenericErrWrapper(netxlite.ECONNRESET)
		const maxBodySize = 1 << 20
		saver := NewSaver()
		v := &SingleHTTPRoundTripValidator{
			ExpectFailure: expectedError,
			ExpectMethod:  "GET",
			ExpectRequestHeaders: map[string][]string{
				"Host":        {"127.0.0.1:8080"},
				"User-Agent":  {"antani/1.0"},
				"X-Client-IP": {"130.192.91.211"},
			},
			ExpectResponseBody:            nil,
			ExpectResponseBodyIsTruncated: false,
			ExpectResponseBodyLength:      0,
			ExpectResponseHeaders:         nil,
			ExpectStatusCode:              0,
			ExpectTransport:               "tcp",
			ExpectURL:                     "http://127.0.0.1:8080/antani",
			RealResponseBody:              nil,
			Saver:                         saver,
		}
		txp := newHTTPTransport(nil, expectedError)
		resp, err := saver.HTTPRoundTrip(txp, maxBodySize, v.NewHTTPRequest())
		if !errors.Is(err, expectedError) {
			t.Fatal(err)
		}
		if resp != nil {
			t.Fatal("expected nil resp")
		}
		if err := v.Validate(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("on failure reading body", func(t *testing.T) {
		expectedError := netxlite.NewTopLevelGenericErrWrapper(netxlite.ECONNRESET)
		const maxBodySize = 1 << 20
		saver := NewSaver()
		v := &SingleHTTPRoundTripValidator{
			ExpectFailure: expectedError,
			ExpectMethod:  "GET",
			ExpectRequestHeaders: map[string][]string{
				"Host":        {"127.0.0.1:8080"},
				"User-Agent":  {"antani/1.0"},
				"X-Client-IP": {"130.192.91.211"},
			},
			ExpectResponseBody:            nil,
			ExpectResponseBodyIsTruncated: false,
			ExpectResponseBodyLength:      0,
			ExpectResponseHeaders: map[string][]string{
				"Server":       {"antani/1.0"},
				"Content-Type": {"text/plain"},
			},
			ExpectStatusCode: 200,
			ExpectTransport:  "tcp",
			ExpectURL:        "http://127.0.0.1:8080/antani",
			RealResponseBody: nil,
			Saver:            saver,
		}
		resp := v.NewHTTPResponse()
		// Hack the body so it returns a connection reset error
		// after some useful piece of data. We do not see any
		// body in the response or in the trace. We may possibly
		// want to include all the body we could read into the
		// trace in the future, but for now it seems fine to do
		// exactly what the previous code was doing.
		resp.Body = io.NopCloser(io.MultiReader(
			bytes.NewReader([]byte("0xdeadbeef")),
			&mocks.Reader{
				MockRead: func(b []byte) (int, error) {
					return 0, expectedError
				},
			},
		))
		txp := newHTTPTransport(resp, nil)
		resp, err := saver.HTTPRoundTrip(txp, maxBodySize, v.NewHTTPRequest())
		if !errors.Is(err, expectedError) {
			t.Fatal("unexpected err", err)
		}
		if resp != nil {
			t.Fatal("expected nil resp")
		}
		if err := v.Validate(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("cloneRequestHeaders", func(t *testing.T) {
		// doWithRequest is an helper function that creates a suitable
		// round tripper and returns trace.HTTPRoundTrip[0] for inspection
		doWithRequest := func(req *http.Request) (*HTTPRoundTripEvent, error) {
			expect := errors.New("mocked err")
			txp := newHTTPTransport(nil, expect)
			saver := NewSaver()
			const maxBodySize = 1 << 20 // irrelevant
			resp, err := saver.HTTPRoundTrip(txp, maxBodySize, req)
			if !errors.Is(err, expect) {
				return nil, fmt.Errorf("unexpected error: %w", err)
			}
			if resp != nil {
				return nil, errors.New("expected nil resp")
			}
			trace := saver.MoveOutTrace()
			if len(trace.HTTPRoundTrip) != 1 {
				return nil, errors.New("expected exactly one HTTPRoundTrip")
			}
			return trace.HTTPRoundTrip[0], nil
		}

		t.Run("with req.URL.Host", func(t *testing.T) {
			req, err := http.NewRequest("GET", "https://x.org/", nil)
			if err != nil {
				t.Fatal(err)
			}
			ev, err := doWithRequest(req)
			if err != nil {
				t.Fatal(err)
			}
			if ev.RequestHeaders.Get("Host") != "x.org" {
				t.Fatal("unexpected request host")
			}
		})

		t.Run("with req.Host", func(t *testing.T) {
			req, err := http.NewRequest("GET", "https://x.org/", nil)
			if err != nil {
				t.Fatal(err)
			}
			req.Host = "google.com"
			ev, err := doWithRequest(req)
			if err != nil {
				t.Fatal(err)
			}
			if ev.RequestHeaders.Get("Host") != "google.com" {
				t.Fatal("unexpected request host")
			}
		})
	})
}

type SingleHTTPRoundTripValidator struct {
	ExpectFailure                 error
	ExpectMethod                  string
	ExpectRequestHeaders          http.Header
	ExpectResponseBody            []byte
	ExpectResponseBodyIsTruncated bool
	ExpectResponseBodyLength      int64
	ExpectResponseHeaders         http.Header
	ExpectStatusCode              int64
	ExpectTransport               string
	ExpectURL                     string
	RealResponseBody              []byte
	Saver                         *Saver
}

func (v *SingleHTTPRoundTripValidator) NewHTTPRequest() *http.Request {
	parsedURL, err := url.Parse(v.ExpectURL)
	runtimex.PanicOnError(err, "url.Parse should not fail here")
	// The saving code clones the headers and adds the host header, which
	// Go would instead add later. So, a realistic mock should not include
	// such an header inside of the http.Request.
	clonedHeaders := v.ExpectRequestHeaders.Clone()
	clonedHeaders.Del("Host")
	return &http.Request{
		Method:           v.ExpectMethod,
		URL:              parsedURL,
		Proto:            "",
		ProtoMajor:       0,
		ProtoMinor:       0,
		Header:           clonedHeaders,
		Body:             nil,
		GetBody:          nil,
		ContentLength:    0,
		TransferEncoding: nil,
		Close:            false,
		Host:             "",
		Form:             nil,
		PostForm:         nil,
		MultipartForm:    nil,
		Trailer:          nil,
		RemoteAddr:       "",
		RequestURI:       "",
		TLS:              nil,
		Cancel:           nil,
		Response:         nil,
	}
}

func (v *SingleHTTPRoundTripValidator) NewHTTPResponse() *http.Response {
	body := io.NopCloser(bytes.NewReader(v.RealResponseBody))
	return &http.Response{
		Status:           http.StatusText(int(v.ExpectStatusCode)),
		StatusCode:       int(v.ExpectStatusCode),
		Proto:            "",
		ProtoMajor:       0,
		ProtoMinor:       0,
		Header:           v.ExpectResponseHeaders,
		Body:             body,
		ContentLength:    0,
		TransferEncoding: nil,
		Close:            false,
		Uncompressed:     false,
		Trailer:          nil,
		Request:          nil,
		TLS:              nil,
	}
}

func (v *SingleHTTPRoundTripValidator) Validate() error {
	trace := v.Saver.MoveOutTrace()
	if len(trace.HTTPRoundTrip) != 1 {
		return errors.New("expected to see one event")
	}
	entry := trace.HTTPRoundTrip[0]
	if !errors.Is(entry.Failure, v.ExpectFailure) {
		return errors.New("unexpected .Failure")
	}
	if !entry.Finished.After(entry.Started) {
		return errors.New(".Finished is not after .Started")
	}
	if entry.Method != v.ExpectMethod {
		return errors.New("unexpected .Method")
	}
	if diff := cmp.Diff(v.ExpectRequestHeaders, entry.RequestHeaders); diff != "" {
		return errors.New(diff)
	}
	if diff := cmp.Diff(v.ExpectResponseBody, entry.ResponseBody); diff != "" {
		return errors.New(diff)
	}
	if entry.ResponseBodyIsTruncated != v.ExpectResponseBodyIsTruncated {
		return errors.New("unexpected .ResponseBodyIsTruncated")
	}
	if entry.ResponseBodyLength != v.ExpectResponseBodyLength {
		return errors.New("unexpected .ResponseBodyLength")
	}
	if diff := cmp.Diff(v.ExpectResponseHeaders, entry.ResponseHeaders); diff != "" {
		return errors.New(diff)
	}
	if entry.StatusCode != v.ExpectStatusCode {
		return errors.New("unexpected .StatusCode")
	}
	if entry.Transport != v.ExpectTransport {
		return errors.New("unexpected .Transport")
	}
	if entry.URL != v.ExpectURL {
		return errors.New("unexpected .URL")
	}
	return nil
}

func TestWrapHTTPTransport(t *testing.T) {
	expected := errors.New("mocked error")
	var txp model.HTTPTransport = &mocks.HTTPTransport{
		MockRoundTrip: func(req *http.Request) (*http.Response, error) {
			return nil, expected
		},
		MockNetwork: func() string {
			return "tcp"
		},
	}
	s := NewSaver()
	txp = s.WrapHTTPTransport(txp, 1<<17)
	resp, err := txp.RoundTrip(&http.Request{
		Method:           "",
		URL:              &url.URL{},
		Proto:            "",
		ProtoMajor:       0,
		ProtoMinor:       0,
		Header:           map[string][]string{},
		Body:             nil,
		ContentLength:    0,
		TransferEncoding: []string{},
		Close:            false,
		Host:             "",
		Form:             map[string][]string{},
		PostForm:         map[string][]string{},
		Trailer:          map[string][]string{},
		RemoteAddr:       "",
		RequestURI:       "",
		Cancel:           make(<-chan struct{}),
		Response:         &http.Response{},
	})
	if !errors.Is(err, expected) {
		t.Fatal("unexpected error", err)
	}
	if resp != nil {
		t.Fatal("expected nil resp")
	}
	mt := s.MoveOutTrace()
	if len(mt.HTTPRoundTrip) != 1 {
		t.Fatal("did not save HTTP round trip")
	}
}
