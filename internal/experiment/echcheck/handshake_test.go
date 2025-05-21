package echcheck

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
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
		want            func(*testing.T, testServerConfig, TestKeys)
	}{
		{
			name:            "no ECH",
			sendGrease:      false,
			useRetryConfigs: false,
			want: func(t *testing.T, testConfig testServerConfig, result TestKeys) {
				if len(result.TCPConnects) != 1 {
					t.Fatal("expected exactly one TCPConnect, got: ", len(result.TCPConnects))
				}
				if len(result.TLSHandshakes) != 1 {
					t.Fatal("expected exactly one TLS handshake, got: ", len(result.TLSHandshakes))
				}
				if result.TLSHandshakes[0].SoError != nil {
					t.Fatal("did not expect soft error, got: ", *result.TLSHandshakes[0].SoError)
				}
				if result.TLSHandshakes[0].Failure != nil {
					t.Fatal("did not expect error, got: ", *result.TLSHandshakes[0].Failure)
				}
				if result.TLSHandshakes[0].OuterServerName != "" {
					t.Fatal("expected OuterServerName to be empty, got: ", result.TLSHandshakes[0].OuterServerName)
				}
			},
		},
		{
			name: "fail to establish ECH handshake",
			// We're using a GREASE ECHConfigList, but we'll handle it as if it's a genuine one (isGrease=False)
			// Test server doesn't handle ECH yet, so it wouldn't send retry configs anyways.
			sendGrease:      true,
			useRetryConfigs: false,
			want: func(t *testing.T, testConfig testServerConfig, result TestKeys) {
				if len(result.TCPConnects) != 1 {
					t.Fatal("expected exactly one TCPConnect, got: ", len(result.TCPConnects))
				}
				if len(result.TLSHandshakes) != 1 {
					t.Fatal("expected exactly one TLS handshake, got: ", len(result.TLSHandshakes))
				}
				if result.TLSHandshakes[0].ServerName != testConfig.url.Hostname() {
					t.Fatal("expected ServerName to be set to ts.URL.Hostname(), got: ", result.TLSHandshakes[0].ServerName)
				}
				if result.TLSHandshakes[0].SoError != nil {
					t.Fatal("did not expect soft error, got: ", *result.TLSHandshakes[0].SoError)
				}
				if result.TLSHandshakes[0].Failure == nil {
					t.Fatal("no GREASE retry so expected Failure to be set")
				}
				if result.TLSHandshakes[0].OuterServerName != testConfig.url.Hostname() {
					t.Fatal("expected OuterServerName to be testserver name, got: ", result.TLSHandshakes[0].OuterServerName)
				}
			},
		},
		{
			name:            "GREASEy ECH handshake",
			sendGrease:      true,
			useRetryConfigs: true,
			want: func(t *testing.T, testConfig testServerConfig, result TestKeys) {
				// Expect 1 TCP Connect and 1 TLS connect because test server does
				// not provide retry configs to initiate a second set.  Upon golang
				// implementing retry_configs, this should become 2 and we should
				// expressly check the 2nd one conforms to expectations.
				if len(result.TCPConnects) != 1 {
					t.Fatal("expected exactly one TCPConnect, got: ", len(result.TCPConnects))
				}
				if len(result.TLSHandshakes) != 1 {
					t.Fatal("expected exactly one TLS handshake, got: ", len(result.TLSHandshakes))
				}
				if result.TLSHandshakes[0].ECHConfig != "GREASE" {
					t.Fatal("expected first ECHConfig to be string literal 'GREASE', got: ", result.TLSHandshakes[0].ECHConfig)
				}
				if result.TLSHandshakes[0].SoError != nil {
					t.Fatal("did not expect soft error, got: ", *result.TLSHandshakes[0].SoError)
				}
				if result.TLSHandshakes[0].Failure != nil {
					t.Fatal("did not expect failure, got: ", *result.TLSHandshakes[0].Failure)
				}
				if result.TLSHandshakes[0].OuterServerName != testConfig.url.Hostname() {
					t.Fatal("expected OuterServerName to be testserver name, got: ", result.TLSHandshakes[0].OuterServerName)
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
			}
			testConfig.tlsConfig.EncryptedClientHelloConfigList = ecl

			ctx := context.Background()
			ch, err := startHandshake(ctx, ecl, test.useRetryConfigs, time.Now(), testConfig.url.Host, testConfig.url, model.DiscardLogger, testConfig.tlsConfig.RootCAs)
			if err != nil {
				t.Fatal(err)
			}
			result := <-ch
			test.want(t, testConfig, result)
		})
	}
}
