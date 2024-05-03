package httpclientx

import "github.com/ooni/probe-cli/v3/internal/model"

// Config contains configuration shared by [GetJSON], [GetXML], [GetRaw], and [PostJSON].
//
// The zero value is invalid; initialize the MANDATORY fields.
type Config struct {
	// Authorization contains the OPTIONAL Authorization header value to use.
	Authorization string

	// Client is the MANDATORY [model.HTTPClient] to use.
	Client model.HTTPClient

	// Logger is the MANDATORY [model.Logger] to use.
	Logger model.Logger

	// MaxResponseBodySize OPTIONALLY limits the maximum body size. If not set, we
	// use the [DefaultMaxResponseBodySize] value.
	MaxResponseBodySize int64

	// UserAgent is the MANDATORY User-Agent header value to use.
	UserAgent string
}

// DefaultMaxResponseBodySize is the default maximum response body size.
const DefaultMaxResponseBodySize = 1 << 24

func (c *Config) maxResponseBodySize() (value int64) {
	value = c.MaxResponseBodySize
	if value <= 0 {
		value = DefaultMaxResponseBodySize
	}
	return
}
