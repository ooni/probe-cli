package netxlite

import (
	"net/http"
	"testing"
)

func TestWrapHTTPClient(t *testing.T) {
	origClient := &http.Client{}
	wrapped := WrapHTTPClient(origClient)
	errWrapper := wrapped.(*httpClientErrWrapper)
	innerClient := errWrapper.HTTPClient.(*http.Client)
	if innerClient != origClient {
		t.Fatal("not the inner client we expected")
	}
}
