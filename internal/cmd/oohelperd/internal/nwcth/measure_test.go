package nwcth

import (
	"context"
	"testing"
)

func TestMeasureWithInvalidURL(t *testing.T) {
	ctx := context.Background()
	req := &ControlRequest{HTTPRequest: "https://google.com", HTTPRequestHeaders: nil}
	_, err := Measure(ctx, req)

	if err == nil {
		t.Fatal("unexpected error")
	}
}
