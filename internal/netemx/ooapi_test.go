package netemx

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

func TestOOAPIHandler(t *testing.T) {
	handler := &OOAPIHandler{}
	server := httptest.NewServer(handler)
	defer server.Close()

	t.Run("for /api/v1/test-helpers with method GET", func(t *testing.T) {
		URL := runtimex.Try1(url.Parse(server.URL))
		URL.Path = "/api/v1/test-helpers"
		resp, err := http.Get(URL.String())
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatal("unexpected status code", resp.StatusCode)
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}

		expectedBody := `{"web-connectivity":[{"address":"https://2.th.ooni.org","type":"https"},{"address":"https://3.th.ooni.org","type":"https"},{"address":"https://0.th.ooni.org","type":"https"},{"address":"https://1.th.ooni.org","type":"https"}]}`

		t.Log(string(body))
		if diff := cmp.Diff([]byte(expectedBody), body); diff != "" {
			t.Fatal(diff)
		}
	})

	t.Run("for unsupported URL path", func(t *testing.T) {
		URL := runtimex.Try1(url.Parse(server.URL))
		URL.Path = "/antani"
		resp, err := http.Get(URL.String())
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusNotFound {
			t.Fatal("unexpected status code", resp.StatusCode)
		}
	})
}
