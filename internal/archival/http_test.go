package archival

import (
	"errors"
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestSaverHTTPRoundTrip(t *testing.T) {
	t.Run("on success without EOF", func(t *testing.T) {})

	t.Run("on success with EOF", func(t *testing.T) {})

	t.Run("on failure during round trip", func(t *testing.T) {})

	t.Run("on failure reading body", func(t *testing.T) {})
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
	Saver                         *Saver
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
