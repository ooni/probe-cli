package backendclient

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ooni/probe-cli/v3/internal/httpapi"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

func newSubmitDescriptor(
	req *model.OOAPICollectorUpdateRequest, reportID string) *httpapi.Descriptor[
	*model.OOAPICollectorUpdateRequest, *model.OOAPICollectorUpdateResponse] {
	rawBody, err := json.Marshal(req)
	runtimex.PanicOnError(err, "json.Marshal failed")
	return &httpapi.Descriptor[*model.OOAPICollectorUpdateRequest, *model.OOAPICollectorUpdateResponse]{
		Accept:             httpapi.ApplicationJSON,
		Authorization:      "",
		AcceptEncodingGzip: false,
		ContentType:        httpapi.ApplicationJSON,
		LogBody:            true,
		MaxBodySize:        0,
		Method:             http.MethodPost,
		Request: &httpapi.RequestDescriptor[*model.OOAPICollectorUpdateRequest]{
			Body: rawBody,
		},
		Response: &httpapi.JSONResponseDescriptor[model.OOAPICollectorUpdateResponse]{},
		Timeout:  0,
		URLPath:  fmt.Sprintf("/report/%s", reportID),
		URLQuery: nil,
	}
}
