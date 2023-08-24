package testingx

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestGeoIPHandlerUbuntu(t *testing.T) {
	handler := &GeoIPHandlerUbuntu{
		ProbeIP: "1.2.3.4",
	}
	server := httptest.NewServer(handler)
	defer server.Close()

	expectBody := []byte(
		`<?xml version="1.0" encoding="UTF-8"?><Response><Ip>1.2.3.4</Ip></Response>`,
	)

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatal("unexpected status code", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	if diff := cmp.Diff(expectBody, body); diff != "" {
		t.Fatal(diff)
	}
}
