package urlgetter_test

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/urlgetter"
	"github.com/ooni/probe-cli/v3/internal/engine/internal/mockable"
	"github.com/ooni/probe-cli/v3/internal/engine/model"
)

func TestMultiIntegration(t *testing.T) {
	multi := urlgetter.Multi{Session: &mockable.Session{}}
	inputs := []urlgetter.MultiInput{{
		Config: urlgetter.Config{Method: "HEAD", NoFollowRedirects: true},
		Target: "https://www.google.com",
	}, {
		Config: urlgetter.Config{Method: "HEAD", NoFollowRedirects: true},
		Target: "https://www.facebook.com",
	}, {
		Config: urlgetter.Config{Method: "HEAD", NoFollowRedirects: true},
		Target: "https://www.kernel.org",
	}, {
		Config: urlgetter.Config{Method: "HEAD", NoFollowRedirects: true},
		Target: "https://www.instagram.com",
	}}
	outputs := multi.Collect(context.Background(), inputs, "integration-test",
		model.NewPrinterCallbacks(log.Log))
	var count int
	for result := range outputs {
		count++
		switch result.Input.Target {
		case "https://www.google.com":
		case "https://www.facebook.com":
		case "https://www.kernel.org":
		case "https://www.instagram.com":
		default:
			t.Fatal("unexpected Input.Target")
		}
		if result.Input.Config.Method != "HEAD" {
			t.Fatal("unexpected Input.Config.Method")
		}
		if result.Err != nil {
			t.Fatal(result.Err)
		}
		if result.TestKeys.Agent != "agent" {
			t.Fatal("invalid TestKeys.Agent")
		}
		if len(result.TestKeys.Queries) != 2 {
			t.Fatal("invalid number of Queries")
		}
		if len(result.TestKeys.Requests) != 1 {
			t.Fatal("invalid number of Requests")
		}
		if len(result.TestKeys.TCPConnect) != 1 {
			t.Fatal("invalid number of TCPConnects")
		}
		if len(result.TestKeys.TLSHandshakes) != 1 {
			t.Fatal("invalid number of TLSHandshakes")
		}
	}
	if count != 4 {
		t.Fatal("invalid number of outputs")
	}
}

func TestMultiIntegrationWithBaseTime(t *testing.T) {
	// We set a beginning of time that's significantly in the past and then
	// fail the test if we see any T smaller than 3600 seconds.
	begin := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	multi := urlgetter.Multi{
		Begin:   begin,
		Session: &mockable.Session{},
	}
	inputs := []urlgetter.MultiInput{{
		Config: urlgetter.Config{Method: "HEAD", NoFollowRedirects: true},
		Target: "https://www.google.com",
	}, {
		Config: urlgetter.Config{Method: "HEAD", NoFollowRedirects: true},
		Target: "https://www.instagram.com",
	}}
	outputs := multi.Collect(context.Background(), inputs, "integration-test",
		model.NewPrinterCallbacks(log.Log))
	var count int
	for result := range outputs {
		for _, entry := range result.TestKeys.NetworkEvents {
			if entry.T < 3600 {
				t.Fatal("base time not correctly set")
			}
			count++
		}
		for _, entry := range result.TestKeys.Queries {
			if entry.T < 3600 {
				t.Fatal("base time not correctly set")
			}
			count++
		}
		for _, entry := range result.TestKeys.TCPConnect {
			if entry.T < 3600 {
				t.Fatal("base time not correctly set")
			}
			count++
		}
		for _, entry := range result.TestKeys.TLSHandshakes {
			if entry.T < 3600 {
				t.Fatal("base time not correctly set")
			}
			count++
		}
	}
	if count <= 0 {
		t.Fatal("unexpected number of entries processed")
	}
}

func TestMultiIntegrationWithoutBaseTime(t *testing.T) {
	// We use the default beginning of time and then fail the test
	// if we see any T smaller than 60 seconds.
	multi := urlgetter.Multi{Session: &mockable.Session{}}
	inputs := []urlgetter.MultiInput{{
		Config: urlgetter.Config{Method: "HEAD", NoFollowRedirects: true},
		Target: "https://www.google.com",
	}, {
		Config: urlgetter.Config{Method: "HEAD", NoFollowRedirects: true},
		Target: "https://www.instagram.com",
	}}
	outputs := multi.Collect(context.Background(), inputs, "integration-test",
		model.NewPrinterCallbacks(log.Log))
	var count int
	for result := range outputs {
		for _, entry := range result.TestKeys.NetworkEvents {
			if entry.T > 60 {
				t.Fatal("base time not correctly set")
			}
			count++
		}
		for _, entry := range result.TestKeys.Queries {
			if entry.T > 60 {
				t.Fatal("base time not correctly set")
			}
			count++
		}
		for _, entry := range result.TestKeys.TCPConnect {
			if entry.T > 60 {
				t.Fatal("base time not correctly set")
			}
			count++
		}
		for _, entry := range result.TestKeys.TLSHandshakes {
			if entry.T > 60 {
				t.Fatal("base time not correctly set")
			}
			count++
		}
	}
	if count <= 0 {
		t.Fatal("unexpected number of entries processed")
	}
}

func TestMultiContextCanceled(t *testing.T) {
	multi := urlgetter.Multi{Session: &mockable.Session{}}
	inputs := []urlgetter.MultiInput{{
		Config: urlgetter.Config{Method: "HEAD", NoFollowRedirects: true},
		Target: "https://www.google.com",
	}, {
		Config: urlgetter.Config{Method: "HEAD", NoFollowRedirects: true},
		Target: "https://www.facebook.com",
	}, {
		Config: urlgetter.Config{Method: "HEAD", NoFollowRedirects: true},
		Target: "https://www.kernel.org",
	}, {
		Config: urlgetter.Config{Method: "HEAD", NoFollowRedirects: true},
		Target: "https://www.instagram.com",
	}}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	outputs := multi.Collect(ctx, inputs, "integration-test",
		model.NewPrinterCallbacks(log.Log))
	var count int
	for result := range outputs {
		count++
		switch result.Input.Target {
		case "https://www.google.com":
		case "https://www.facebook.com":
		case "https://www.kernel.org":
		case "https://www.instagram.com":
		default:
			t.Fatal("unexpected Input.Target")
		}
		if result.Input.Config.Method != "HEAD" {
			t.Fatal("unexpected Input.Config.Method")
		}
		if !errors.Is(result.Err, context.Canceled) {
			t.Fatal("unexpected error")
		}
		if result.TestKeys.Agent != "agent" {
			t.Fatal("invalid TestKeys.Agent")
		}
		if len(result.TestKeys.Queries) != 0 {
			t.Fatal("invalid number of Queries")
		}
		if len(result.TestKeys.Requests) != 1 {
			t.Fatal("invalid number of Requests")
		}
		if len(result.TestKeys.TCPConnect) != 0 {
			t.Fatal("invalid number of TCPConnects")
		}
		if len(result.TestKeys.TLSHandshakes) != 0 {
			t.Fatal("invalid number of TLSHandshakes")
		}
	}
	if count != 4 {
		t.Fatal("invalid number of outputs")
	}
}

func TestMultiWithSpecificCertPool(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello, client")
	}))
	defer server.Close()
	cert := server.Certificate()
	certpool := x509.NewCertPool()
	certpool.AddCert(cert)
	multi := urlgetter.Multi{Session: &mockable.Session{}}
	inputs := []urlgetter.MultiInput{{
		Config: urlgetter.Config{
			CertPool:          certpool,
			Method:            "GET",
			NoFollowRedirects: true,
		},
		Target: server.URL,
	}}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	outputs := multi.Collect(ctx, inputs, "integration-test",
		model.NewPrinterCallbacks(log.Log))
	var count int
	for result := range outputs {
		count++
		if result.Err != nil {
			t.Fatal(result.Err)
		}
	}
	if count != 1 {
		t.Fatal("unexpected count value")
	}
}
