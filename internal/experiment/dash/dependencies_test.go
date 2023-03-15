package dash

//
// Mockable implementations for testing.
//

import (
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// mockableDependencies mocks the [dependencies] of each dash phase.
type mockableDependencies struct {
	MockHTTPClient func() model.HTTPClient

	MockNewHTTPRequestWithContext func(
		ctx context.Context, method, url string, body io.Reader) (*http.Request, error)

	MockReadAllContext func(ctx context.Context, body io.Reader) ([]byte, error)
}

var _ dependencies = &mockableDependencies{}

// HTTPClient implements [dependencies].
func (d *mockableDependencies) HTTPClient() model.HTTPClient {
	return d.MockHTTPClient()
}

// Logger implements [dependencies].
func (d *mockableDependencies) Logger() model.Logger {
	return model.DiscardLogger
}

// NewHTTPRequestWithContext implements [dependencies].
func (d *mockableDependencies) NewHTTPRequestWithContext(
	ctx context.Context, method string, url string, body io.Reader) (*http.Request, error) {
	return d.MockNewHTTPRequestWithContext(ctx, method, url, body)
}

// ReadAllContext implements [dependencies].
func (d *mockableDependencies) ReadAllContext(ctx context.Context, r io.Reader) ([]byte, error) {
	return d.MockReadAllContext(ctx, r)
}

// UserAgent implements [dependencies].
func (d *mockableDependencies) UserAgent() string {
	return "miniooni/0.1.0-dev"
}

func TestMockableDependencies(t *testing.T) {
	// nothing
}
