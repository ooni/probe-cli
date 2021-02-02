// Package httpfailure groups a bunch of extra HTTP failures.
//
// These failures only matter in the context of processing the results
// of specific experiments, e.g., whatsapp, telegram.
package httpfailure

var (
	// UnexpectedStatusCode indicates that we re not getting
	// the expected (range of) HTTP status code(s).
	UnexpectedStatusCode = "http_unexpected_status_code"

	// UnexpectedRedirectURL indicates that the redirect URL
	// returned by the server is not the expected one.
	UnexpectedRedirectURL = "http_unexpected_redirect_url"
)
