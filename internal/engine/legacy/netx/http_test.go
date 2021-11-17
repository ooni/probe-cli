package netx_test

import (
	"context"
	"crypto/x509"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine/legacy/netx"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func dowithclient(t *testing.T, client *netx.HTTPClient) {
	defer client.CloseIdleConnections()
	resp, err := client.HTTPClient.Get("https://www.google.com")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	_, err = netxlite.ReadAllContext(context.Background(), resp.Body)
	if err != nil {
		t.Fatal(err)
	}
}

func TestHTTPClient(t *testing.T) {
	client := netx.NewHTTPClient()
	dowithclient(t, client)
}

func TestHTTPClientAndTransport(t *testing.T) {
	client := netx.NewHTTPClient()
	client.Transport = netx.NewHTTPTransport()
	dowithclient(t, client)
}

func TestHTTPClientConfigureDNS(t *testing.T) {
	client := netx.NewHTTPClientWithoutProxy()
	err := client.ConfigureDNS("udp", "1.1.1.1:53")
	if err != nil {
		t.Fatal(err)
	}
	dowithclient(t, client)
}

func TestHTTPClientSetResolver(t *testing.T) {
	client := netx.NewHTTPClientWithoutProxy()
	client.SetResolver(new(net.Resolver))
	dowithclient(t, client)
}

func TestHTTPClientSetCABundle(t *testing.T) {
	client := netx.NewHTTPClientWithoutProxy()
	err := client.SetCABundle("testdata/cacert.pem")
	if err != nil {
		t.Fatal(err)
	}
	resp, err := client.HTTPClient.Get("https://www.google.com")
	var target x509.UnknownAuthorityError
	if errors.As(err, &target) == false {
		t.Fatal("not the error we expected")
	}
	if resp != nil {
		t.Fatal("expected a nil conn here")
	}
}

func TestHTTPClientForceSpecificSNI(t *testing.T) {
	client := netx.NewHTTPClientWithoutProxy()
	err := client.ForceSpecificSNI("www.facebook.com")
	if err != nil {
		t.Fatal(err)
	}
	resp, err := client.HTTPClient.Get("https://www.google.com")
	var target x509.HostnameError
	if errors.As(err, &target) == false {
		t.Fatal("not the error we expected")
	}
	if resp != nil {
		t.Fatal("expected a nil response here")
	}
}

func TestHTTPClientForceSkipVerify(t *testing.T) {
	client := netx.NewHTTPClientWithoutProxy()
	client.ForceSkipVerify()
	resp, err := client.HTTPClient.Get("https://self-signed.badssl.com/")
	if err != nil {
		t.Fatal(err)
	}
	if resp == nil {
		t.Fatal("expected non nil response here")
	}
}

func TestHTTPNewClientProxy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(451)
		}))
	defer server.Close()
	client := netx.NewHTTPClientWithoutProxy()
	httpProxyTestMain(t, client.HTTPClient, 200)
	client = netx.NewHTTPClientWithProxyFunc(func(req *http.Request) (*url.URL, error) {
		return url.Parse(server.URL)
	})
	httpProxyTestMain(t, client.HTTPClient, 451)
}

const httpProxyTestsURL = "http://explorer.ooni.org"

func httpProxyTestMain(t *testing.T, client *http.Client, expect int) {
	req, err := http.NewRequest("GET", httpProxyTestsURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	_, err = netxlite.ReadAllContext(context.Background(), resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != expect {
		t.Fatal("unexpected status code")
	}
}

func TestHTTPTransportTimeout(t *testing.T) {
	client := &http.Client{Transport: netx.NewHTTPTransport()}
	req, err := http.NewRequest("GET", "https://www.google.com", nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	req = req.WithContext(ctx)
	resp, err := client.Do(req)
	if err == nil {
		t.Fatal("expected an error here")
	}
	if !strings.HasSuffix(err.Error(), netxlite.FailureGenericTimeoutError) {
		t.Fatal("not the error we expected")
	}
	if resp != nil {
		t.Fatal("expected nil resp here")
	}
}

func TestHTTPTransportFailure(t *testing.T) {
	client := &http.Client{Transport: netx.NewHTTPTransport()}
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
