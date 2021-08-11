package nwcth

import (
	"io"
	"net/http"
)

func HTTPDo(req *http.Request, transport http.RoundTripper) *HTTPRoundtripMeasurement {
	httpRoundtrip := &HTTPRoundtripMeasurement{
		Request: &HTTPRequest{
			Headers: req.Header,
		},
	}
	clnt := http.Client{
		CheckRedirect: func(r *http.Request, reqs []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Transport: transport,
	}
	resp, err := clnt.Do(req)
	if err != nil {
		s := err.Error()
		httpRoundtrip.Response = &HTTPResponse{
			Failure: &s,
		}
		return httpRoundtrip
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	httpRoundtrip.Response = &HTTPResponse{
		BodyLength: int64(len(body)),
		Failure:    nil,
		Headers:    resp.Header,
		StatusCode: int64(resp.StatusCode),
	}
	return httpRoundtrip
}
