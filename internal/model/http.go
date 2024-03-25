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
	// is 28.39% of the browser population as of 2023-12-13 according to the
	// https://www.useragents.me/ webpage.
	HTTPHeaderUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.3"
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
