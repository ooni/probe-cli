package oldhttptransport

import (
	"context"
	"net/http"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/engine/legacy/netx/transactionid"
	"github.com/ooni/probe-cli/v3/internal/iox"
)

type transactionerCheckTransactionID struct {
	roundTripper http.RoundTripper
	t            *testing.T
}

func (t *transactionerCheckTransactionID) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := req.Context()
	if id := transactionid.ContextTransactionID(ctx); id == 0 {
		t.t.Fatal("transaction ID not set")
	}
	return t.roundTripper.RoundTrip(req)
}

func TestTransactionerSuccess(t *testing.T) {
	client := &http.Client{
		Transport: NewTransactioner(&transactionerCheckTransactionID{
			roundTripper: http.DefaultTransport,
			t:            t,
		}),
	}
	resp, err := client.Get("https://www.google.com")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	_, err = iox.ReadAllContext(context.Background(), resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	client.CloseIdleConnections()
}

func TestTransactionerFailure(t *testing.T) {
	client := &http.Client{
		Transport: NewTransactioner(http.DefaultTransport),
	}
	// This fails the request because we attempt to speak cleartext HTTP with
	// a server that instead is expecting TLS.
	resp, err := client.Get("http://www.google.com:443")
	if err == nil {
		t.Fatal("expected an error here")
	}
	if resp != nil {
		t.Fatal("expected a nil response here")
	}
	client.CloseIdleConnections()
}
