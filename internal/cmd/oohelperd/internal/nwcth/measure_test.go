package nwcth

import (
	"context"
	"strings"
	"testing"
)

func TestMeasureWithInvalidURL(t *testing.T) {
	ctx := context.Background()
	req := &CtrlRequest{HTTPRequest: "http://[::1]aaaa", HTTPRequestHeaders: nil}
	_, err := Measure(ctx, req)

	if err == nil || !strings.HasSuffix(err.Error(), `invalid port "aaaa" after host`) {
		t.Fatal("expected an error here")
	}
}
