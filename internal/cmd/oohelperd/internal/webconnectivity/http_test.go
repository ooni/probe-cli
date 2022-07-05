package webconnectivity

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func TestHTTPDoWithInvalidURL(t *testing.T) {
	ctx := context.Background()
	wg := new(sync.WaitGroup)
	httpch := make(chan CtrlHTTPResponse, 1)
	wg.Add(1)
	go HTTPDo(ctx, &HTTPConfig{
		Headers:           nil,
		MaxAcceptableBody: 1 << 24,
		NewClient: func() model.HTTPClient {
			return http.DefaultClient
		},
		Out: httpch,
		URL: "http://[::1]aaaa",
		Wg:  wg,
	})
	// wait for measurement steps to complete
	wg.Wait()
	resp := <-httpch
	if resp.Failure == nil || *resp.Failure != "unknown_error" {
		t.Fatal("not the failure we expected")
	}
}

func TestHTTPDoWithHTTPTransportFailure(t *testing.T) {
	expected := errors.New("mocked error")
	ctx := context.Background()
	wg := new(sync.WaitGroup)
	httpch := make(chan CtrlHTTPResponse, 1)
	wg.Add(1)
	go HTTPDo(ctx, &HTTPConfig{
		Headers:           nil,
		MaxAcceptableBody: 1 << 24,
		NewClient: func() model.HTTPClient {
			return &http.Client{
				Transport: FakeTransport{
					Err: expected,
				},
			}
		},
		Out: httpch,
		URL: "http://www.x.org",
		Wg:  wg,
	})
	// wait for measurement steps to complete
	wg.Wait()
	resp := <-httpch
	if resp.Failure == nil || *resp.Failure != "unknown_error" {
		t.Fatal("not the error we expected")
	}
}

func newErrWrapper(failure, operation string) error {
	return &netxlite.ErrWrapper{
		Failure:    failure,
		Operation:  operation,
		WrappedErr: nil, // should not matter
	}
}

func newErrWrapperTopLevel(failure string) error {
	return newErrWrapper(failure, netxlite.TopLevelOperation)
}

func Test_httpMapFailure(t *testing.T) {
	tests := []struct {
		name    string
		failure error
		want    *string
	}{{
		name:    "nil",
		failure: nil,
		want:    nil,
	}, {
		name:    "nxdomain",
		failure: newErrWrapperTopLevel(netxlite.FailureDNSNXDOMAINError),
		want:    stringPointerForString("dns_lookup_error"),
	}, {
		name:    "no answer",
		failure: newErrWrapperTopLevel(netxlite.FailureDNSNoAnswer),
		want:    stringPointerForString("dns_lookup_error"),
	}, {
		name:    "non recoverable failure",
		failure: newErrWrapperTopLevel(netxlite.FailureDNSNonRecoverableFailure),
		want:    stringPointerForString("dns_lookup_error"),
	}, {
		name:    "refused",
		failure: newErrWrapperTopLevel(netxlite.FailureDNSRefusedError),
		want:    stringPointerForString("dns_lookup_error"),
	}, {
		name:    "server misbehaving",
		failure: newErrWrapperTopLevel(netxlite.FailureDNSServerMisbehaving),
		want:    stringPointerForString("dns_lookup_error"),
	}, {
		name:    "temporary failure",
		failure: newErrWrapperTopLevel(netxlite.FailureDNSTemporaryFailure),
		want:    stringPointerForString("dns_lookup_error"),
	}, {
		name:    "timeout outside of dns lookup",
		failure: newErrWrapperTopLevel(netxlite.FailureGenericTimeoutError),
		want:    stringPointerForString(netxlite.FailureGenericTimeoutError),
	}, {
		name:    "timeout inside of dns lookup",
		failure: newErrWrapper(netxlite.FailureGenericTimeoutError, netxlite.ResolveOperation),
		want:    stringPointerForString("dns_lookup_error"),
	}, {
		name:    "connection refused",
		failure: newErrWrapperTopLevel(netxlite.FailureConnectionRefused),
		want:    stringPointerForString("connection_refused_error"),
	}, {
		name:    "anything else",
		failure: newErrWrapperTopLevel(netxlite.FailureEOFError),
		want:    stringPointerForString("unknown_error"),
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := httpMapFailure(tt.failure)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
