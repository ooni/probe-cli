package testingproxy

import "net/http"

type httpClient interface {
	Get(URL string) (*http.Response, error)
}

type httpClientMock struct {
	MockGet func(URL string) (*http.Response, error)
}

var _ httpClient = &httpClientMock{}

// Get implements httpClient.
func (c *httpClientMock) Get(URL string) (*http.Response, error) {
	return c.MockGet(URL)
}

type httpTestingT interface {
	Logf(format string, v ...any)
	Fatal(v ...any)
}

type httpTestingTMock struct {
	MockLogf  func(format string, v ...any)
	MockFatal func(v ...any)
}

var _ httpTestingT = &httpTestingTMock{}

// Fatal implements httpTestingT.
func (t *httpTestingTMock) Fatal(v ...any) {
	t.MockFatal(v...)
}

// Logf implements httpTestingT.
func (t *httpTestingTMock) Logf(format string, v ...any) {
	t.MockLogf(format, v...)
}

func httpCheckResponse(t httpTestingT, client httpClient, targetURL string) {
	resp, err := client.Get(targetURL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	t.Logf("%+v", resp)
	if resp.StatusCode != 200 {
		t.Fatal("invalid status code")
	}
}
