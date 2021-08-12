package nwcth

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"testing"
)

func TestMeasureSuccess(t *testing.T) {
	req := &ControlRequest{
		HTTPRequest: "https://example.com",
	}
	resp, err := Measure(context.Background(), req, &Config{})
	if err != nil {
		t.Fatal("unexpected error")
	}
	if resp == nil {
		t.Fatal("unexpected nil response")
	}
}

type MockChecker struct{}

func (c *MockChecker) InitialChecks(URL string) (*url.URL, error) {
	return nil, ErrExpectedCheck
}

type MockExplorer struct{}

func (c *MockExplorer) Explore(URL *url.URL) ([]*RoundTrip, error) {
	return nil, ErrExpectedExplore
}

type MockGenerator struct{}

func (c *MockGenerator) Generate(ctx context.Context, rts []*RoundTrip) ([]*URLMeasurement, error) {
	return nil, ErrExpectedGenerate
}

var ErrExpectedCheck error = errors.New("expected error checker")
var ErrExpectedExplore error = errors.New("expected error explorer")
var ErrExpectedGenerate error = errors.New("expected error generator")

func TestMeasureInitialChecksFail(t *testing.T) {
	req := &ControlRequest{
		HTTPRequest: "https://example.com",
	}
	resp, err := Measure(context.Background(), req, &Config{checker: &MockChecker{}})
	if err == nil {
		t.Fatal("expected an error here")
	}
	if err != ErrExpectedCheck {
		t.Fatal("unexpected error type")
	}
	if resp == nil {
		t.Fatal("resp should not be nil")
	}
	if len(resp.URLMeasurements) != 1 {
		t.Fatal("unexpected number of measurements")
	}
	if resp.URLMeasurements[0].DNS == nil {
		t.Fatal("DNS entry should not be nil")
	}
	if !strings.HasSuffix(*resp.URLMeasurements[0].DNS.Failure, ErrExpectedCheck.Error()) {
		t.Fatal("unexpected failure")
	}
}

func TestMeasureExploreFails(t *testing.T) {
	req := &ControlRequest{
		HTTPRequest: "https://example.com",
	}
	resp, err := Measure(context.Background(), req, &Config{explorer: &MockExplorer{}})
	if err == nil {
		t.Fatal("expected an error here")
	}
	if err != ErrExpectedExplore {
		t.Fatal("unexpected error type")
	}
	if resp == nil {
		t.Fatal("resp should not be nil")
	}
	if len(resp.URLMeasurements) != 1 {
		t.Fatal("unexpected number of measurements")
	}
	if resp.URLMeasurements[0].DNS == nil {
		t.Fatal("DNS entry should not be nil")
	}
	if resp.URLMeasurements[0].DNS.Failure != nil {
		t.Fatal("unexpected DNS failure")
	}
}

func TestMeasureGenerateFails(t *testing.T) {
	req := &ControlRequest{
		HTTPRequest: "https://example.com",
	}
	resp, err := Measure(context.Background(), req, &Config{generator: &MockGenerator{}})
	if err == nil {
		t.Fatal("expected an error here")
	}
	if err != ErrExpectedGenerate {
		t.Fatal("unexpected error type")
	}
	if resp != nil {
		t.Fatal("resp should be nil")
	}
}
