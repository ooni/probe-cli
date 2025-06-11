package model

//
// Common HTTP definitions.
//

// Headers we use for measuring.
const (
	// HTTPHeaderAccept is the Accept header used for measuring.
	HTTPHeaderAccept = "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8"

	// HTTPHeaderAcceptLanguage is the Accept-Language header used for measuring.
	HTTPHeaderAcceptLanguage = "en-US,en;q=0.9"

	// HTTPHeaderUserAgent is the User-Agent header used for measuring. The current header
	// is 40.98% of the browser population as of 2025-02-09 according to the
	// https://www.useragents.me/ webpage.
	HTTPHeaderUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.10 Safari/605.1.1"
)

// Additional strings used to report HTTP errors. They're currently only used by
// experiment/whatsapp but may be used by more experiments in the future. They must
// be addressable (i.e., var and not const) because experiments typically want to
// take their addresses to fill fields with `string|null` type.
var (
	// HTTPUnexpectedStatusCode indicates that we re not getting
	// the expected (range of) HTTP status code(s).
	HTTPUnexpectedStatusCode = "http_unexpected_status_code"

	// HTTPUnexpectedRedirectURL indicates that the redirect URL
	// returned by the server is not the expected one.
	HTTPUnexpectedRedirectURL = "http_unexpected_redirect_url"
)
