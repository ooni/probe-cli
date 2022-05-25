package model

//
// Common HTTP definitions and errors.
//

const (
	// HTTPHeaderAccept is the Accept header used for measuring.
	HTTPHeaderAccept = "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8"

	// HTTPHeaderAcceptLanguage is the Accept-Language header used for measuring.
	HTTPHeaderAcceptLanguage = "en-US,en;q=0.9"

	// HTTPHeaderUserAgent is the User-Agent header used for measuring.
	HTTPHeaderUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/100.0.4896.127 Safari/537.36"
)

var (
	// HTTPUnexpectedStatusCode indicates that we re not getting
	// the expected (range of) HTTP status code(s).
	HTTPUnexpectedStatusCode = "http_unexpected_status_code"

	// HTTPUnexpectedRedirectURL indicates that the redirect URL
	// returned by the server is not the expected one.
	HTTPUnexpectedRedirectURL = "http_unexpected_redirect_url"
)
