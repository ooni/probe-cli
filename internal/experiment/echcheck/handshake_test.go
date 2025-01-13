package echcheck

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
)

type testServerConfig struct {
	ts        *httptest.Server
	url       *url.URL
	tlsConfig *tls.Config
}

func setupTest(t *testing.T) testServerConfig {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "success")
	}))

	parsed, err := url.Parse(ts.URL)
	if err != nil {
		t.Fatal(err)
	}

	testPool := x509.NewCertPool()
	testPool.AddCert(ts.Certificate())
	tlsConfig := &tls.Config{
		ServerName: parsed.Hostname(),
		RootCAs:    testPool,
	}

	return testServerConfig{
		ts:        ts,
		url:       parsed,
		tlsConfig: tlsConfig,
	}
}

func TestHandshake(t *testing.T) {
	tests := []struct {
		name            string
		sendGrease      bool
		useRetryConfigs bool
		want            func(*testing.T, testServerConfig, *model.ArchivalTLSOrQUICHandshakeResult)
	}{
		{
			name:            "no ECH",
			sendGrease:      false,
			useRetryConfigs: false,
			want: func(t *testing.T, testConfig testServerConfig, result *model.ArchivalTLSOrQUICHandshakeResult) {
				if result.SoError != nil {
					t.Fatal("did not expect error, got: ", result.SoError)
				}
				if result.Failure != nil {
					t.Fatal("did not expect error, got: ", *result.Failure)
				}
				if result.OuterServerName != "" {
					t.Fatal("expected OuterServerName to be empty, got: ", result.OuterServerName)
				}
			},
		},
		{
			name: "fail to establish ECH handshake",
			// We're using a GREASE ECHConfigList, but we'll handle it as if it's a genuine one (isGrease=False)
			// Test server doesn't handle ECH yet, so it wouldn't send retry configs anyways.
			sendGrease:      true,
			useRetryConfigs: false,
			want: func(t *testing.T, testConfig testServerConfig, result *model.ArchivalTLSOrQUICHandshakeResult) {
				if result.ServerName != testConfig.url.Hostname() {
					t.Fatal("expected ServerName to be set to ts.URL.Hostname(), got: ", result.ServerName)
				}

				if result.SoError != nil {
					t.Fatal("did not expect error, got: ", result.SoError)
				}

				if result.Failure == nil || !strings.Contains(*result.Failure, "tls: server rejected ECH") {
					t.Fatal("server should have rejected ECH: ", *result.Failure)
				}
			},
		},
		{
			name:            "GREASEy ECH handshake",
			sendGrease:      true,
			useRetryConfigs: true,
			want: func(t *testing.T, testConfig testServerConfig, result *model.ArchivalTLSOrQUICHandshakeResult) {
				if result.ECHConfig != "GREASE" {
					t.Fatal("expected ECHConfig to be string literal 'GREASE', got: ", result.ECHConfig)
				}
				if result.SoError != nil {
					t.Fatal("did not expect error, got: ", result.SoError)
				}
				if result.Failure == nil || !strings.Contains(*result.Failure, "tls: server rejected ECH") {
					t.Fatal("expected Connection to fail because test server doesn't handle ECH yet")
				}
			},
		},
		// TODO: Add a test case with Real ECH once the server-side of crypto/tls supports it.
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			testConfig := setupTest(t)
			defer testConfig.ts.Close()

			ecl := []byte{}
			if test.sendGrease {
				grease, err := generateGreaseyECHConfigList(rand.Reader, testConfig.url.Hostname())
				if err != nil {
					t.Fatal(err)
				}
				ecl = grease
				testConfig.tlsConfig.EncryptedClientHelloConfigList = ecl
			}

			ctx := context.Background()
			conn, err := net.Dial("tcp", testConfig.url.Host)
			if err != nil {
				t.Fatal(err)
			}
			result := handshake(ctx, conn, ecl, test.useRetryConfigs, time.Now(), testConfig.url.Host, model.DiscardLogger, testConfig.tlsConfig)
			test.want(t, testConfig, result)
		})
	}
}
