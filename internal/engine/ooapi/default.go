package ooapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"text/template"
)

type defaultRequestMaker struct{}

func (*defaultRequestMaker) NewRequest(
	ctx context.Context, method, URL string, body io.Reader) (*http.Request, error) {
	return http.NewRequestWithContext(ctx, method, URL, body)
}

type defaultJSONCodec struct{}

func (*defaultJSONCodec) Encode(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func (*defaultJSONCodec) Decode(b []byte, v interface{}) error {
	return json.Unmarshal(b, v)
}

type defaultTemplateExecutor struct{}

func (*defaultTemplateExecutor) Execute(tmpl string, v interface{}) (string, error) {
	to, err := template.New("t").Parse(tmpl)
	if err != nil {
		return "", err
	}
	var sb strings.Builder
	if err := to.Execute(&sb, v); err != nil {
		return "", err
	}
	return sb.String(), nil
}
