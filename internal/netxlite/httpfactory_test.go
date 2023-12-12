package netxlite

import (
	"net/url"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/feature/oohttpfeat"
	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func TestNewHTTPTransportWithOptions(t *testing.T) {

	t.Run("make sure that we get the correct types and settings", func(t *testing.T) {
		expectDialer := &mocks.Dialer{}
		expectTLSDialer := &mocks.TLSDialer{}
		expectLogger := model.DiscardLogger
		txp := NewHTTPTransportWithOptions(expectLogger, expectDialer, expectTLSDialer)

		// undo the results of the netxlite.WrapTransport function
		txpLogger := txp.(*httpTransportLogger)
		if txpLogger.Logger != expectLogger {
			t.Fatal("invalid logger")
		}
		txpErrWrapper := txpLogger.HTTPTransport.(*httpTransportErrWrapper)

		// make sure we correctly configured dialer and TLS dialer
		txpCloser := txpErrWrapper.HTTPTransport.(*httpTransportConnectionsCloser)
		timeoutDialer := txpCloser.Dialer.(*httpDialerWithReadTimeout)
		childDialer := timeoutDialer.Dialer
		if childDialer != expectDialer {
			t.Fatal("invalid dialer")
		}
		timeoutTLSDialer := txpCloser.TLSDialer.(*httpTLSDialerWithReadTimeout)
		childTLSDialer := timeoutTLSDialer.TLSDialer
		if childTLSDialer != expectTLSDialer {
			t.Fatal("invalid TLS dialer")
		}

		// make sure there's the stdlib adapter
		stdlibAdapter := txpCloser.HTTPTransport.(*httpTransportStdlib)
		underlying := stdlibAdapter.StdlibTransport

		// finish checking by explicitly inspecting the fields we modify
		if underlying.GetDialContext() == nil {
			t.Fatal("expected non-nil .DialContext")
		}
		if underlying.GetDialTLSContext() == nil {
			t.Fatal("expected non-nil .DialTLSContext")
		}
		if underlying.GetProxy() != nil {
			t.Fatal("expected nil .Proxy")
		}
		if underlying.GetForceAttemptHTTP2() != oohttpfeat.ExpectedForceAttemptHTTP2 {
			t.Fatal("expected true .ForceAttemptHTTP2")
		}
		if !underlying.GetDisableCompression() {
			t.Fatal("expected true .DisableCompression")
		}
	})

	unwrap := func(txp model.HTTPTransport) *oohttpfeat.HTTPTransport {
		txpLogger := txp.(*httpTransportLogger)
		txpErrWrapper := txpLogger.HTTPTransport.(*httpTransportErrWrapper)
		txpCloser := txpErrWrapper.HTTPTransport.(*httpTransportConnectionsCloser)
		stdlibAdapter := txpCloser.HTTPTransport.(*httpTransportStdlib)
		oohttpStdlibAdapter := stdlibAdapter.StdlibTransport
		return oohttpStdlibAdapter
	}

	t.Run("make sure HTTPTransportOptionProxyURL is WAI", func(t *testing.T) {
		runWithURL := func(expectedURL *url.URL) {
			expectDialer := &mocks.Dialer{}
			expectTLSDialer := &mocks.TLSDialer{}
			expectLogger := model.DiscardLogger
			txp := NewHTTPTransportWithOptions(
				expectLogger,
				expectDialer,
				expectTLSDialer,
				HTTPTransportOptionProxyURL(expectedURL),
			)
			underlying := unwrap(txp)
			proxy := underlying.GetProxy()
			if proxy == nil {
				t.Fatal("expected non-nil .Proxy")
			}
			got, err := proxy(&oohttpfeat.HTTPRequest{})
			if err != nil {
				t.Fatal(err)
			}
			if got != expectedURL {
				t.Fatal("not the expected URL")
			}
		}

		runWithURL(&url.URL{})

		runWithURL(nil)
	})

	t.Run("make sure HTTPTransportOptionMaxConnsPerHost is WAI", func(t *testing.T) {
		runWithValue := func(expectedValue int) {
			expectDialer := &mocks.Dialer{}
			expectTLSDialer := &mocks.TLSDialer{}
			expectLogger := model.DiscardLogger
			txp := NewHTTPTransportWithOptions(
				expectLogger,
				expectDialer,
				expectTLSDialer,
				HTTPTransportOptionMaxConnsPerHost(expectedValue),
			)
			underlying := unwrap(txp)
			got := underlying.GetMaxConnsPerHost()
			if got != expectedValue {
				t.Fatal("not the expected value")
			}
		}

		runWithValue(100)

		runWithValue(10)
	})

	t.Run("make sure HTTPTransportDisableCompression is WAI", func(t *testing.T) {
		runWithValue := func(expectedValue bool) {
			expectDialer := &mocks.Dialer{}
			expectTLSDialer := &mocks.TLSDialer{}
			expectLogger := model.DiscardLogger
			txp := NewHTTPTransportWithOptions(
				expectLogger,
				expectDialer,
				expectTLSDialer,
				HTTPTransportOptionDisableCompression(expectedValue),
			)
			underlying := unwrap(txp)
			got := underlying.GetDisableCompression()
			if got != expectedValue {
				t.Fatal("not the expected value")
			}
		}

		runWithValue(true)

		runWithValue(false)
	})
}
