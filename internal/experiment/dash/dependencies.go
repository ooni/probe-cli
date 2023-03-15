package dash

//
// Dependencies for unit testing.
//

import (
	"context"
	"io"
	"net/http"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// dependencies allows unit testing each phase of the experiment.
type dependencies interface {
	// HTTPClient returns the HTTP client to use.
	HTTPClient() model.HTTPClient

	// Logger returns the logger we should use.
	Logger() model.Logger

	// NewHTTPRequestWithContext allows to mock the [http.NewRequestWithContext] function.
	NewHTTPRequestWithContext(
		context context.Context, method string, url string, body io.Reader) (*http.Request, error)

	// UserAgent returns the user agent we should use.
	UserAgent() string
}
