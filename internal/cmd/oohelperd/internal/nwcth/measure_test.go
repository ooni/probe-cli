package nwcth

import (
	"context"
	"testing"
)

func TestMeasureWithInvalidURL(t *testing.T) {
	ctx := context.Background()
	req := &ControlRequest{HTTPRequest: "http://[::1]aaaa", HTTPRequestHeaders: nil}
	_, err := Measure(ctx, req)

	if err == nil || err != ErrInvalidURL {
		t.Fatal("expected an error here", err.Error())
	}
}
