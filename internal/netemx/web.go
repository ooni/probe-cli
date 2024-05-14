package netemx

import (
	"net"
	"net/http"

	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// ExampleWebPage is the webpage returned by [ExampleWebPageHandlerFactory].
const ExampleWebPage = `<!doctype html>
<html>
<head>
	<title>Default Web Page</title>
</head>
<body>
<div>
	<h1>Default Web Page</h1>

	<p>This is the default web page of the default domain.</p>

	<p>We detect webpage blocking by checking for the status code first. If the status
	code is different, we consider the measurement http-diff. On the contrary when
	the status code matches, we say it's all good if one of the following check succeeds:</p>

	<p><ol>
		<li>the body length does not match (we say they match is the smaller of the two
		webpages is 70% or more of the size of the larger webpage);</li>

		<li>the uncommon headers match;</li>

		<li>the webpage title contains mostly the same words.</li>
	</ol></p>

	<p>If the three above checks fail, then we also say that there is http-diff. Because
	we need QA checks to work as intended, the size of THIS webpage you are reading
	has been increased, by adding this description, such that the body length check fails. The
	original webpage size was too close to the blockpage in size, and therefore we did see
	that there was no http-diff, as it ought to be.</p>

	<p>To make sure we're not going to have this issue in the future, there is now a runtime
	check that causes our code to crash if this web page size is too similar to the one of
	the default blockpage. We chose to add this text for additional clarity.</p>

	<p>Also, note that the blockpage MUST be very small, because in some cases we need
	to spoof it into a single TCP segment using ooni/netem's DPI.</p>
</div>
</body>
</html>
`

func init() {
	ratio := float64(len(Blockpage)) / float64(len(ExampleWebPage))
	runtimex.Assert(ratio < 0.7, "The ExampleWebPage is too short and would be confused with the Blockpage")
}

// ExampleWebPageHandler returns a handler returning a webpage similar to example.org's one when the domain
// is www.example.{com,org} and redirecting to www. when the domain is example.{com,org}.
func ExampleWebPageHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Alt-Svc", `h3=":443"`)
		w.Header().Add("Date", "Thu, 24 Aug 2023 14:35:29 GMT")

		// According to Go documentation, the host header is removed from the
		// header fields and included as (*Request).Host
		//
		// Empirically, this field could either contain an host name or it could
		// be an endpoint, i.e., it could also contain an optional port
		host := r.Host
		if h, _, err := net.SplitHostPort(host); err == nil {
			host = h
		}

		switch host {
		case "www.example.com", "www.example.org":
			_, _ = w.Write([]byte(ExampleWebPage))

		case "example.com":
			w.Header().Add("Location", "https://www.example.com/")
			w.WriteHeader(http.StatusPermanentRedirect)

		case "example.org":
			w.Header().Add("Location", "https://www.example.org/")
			w.WriteHeader(http.StatusPermanentRedirect)

		default:
			w.WriteHeader(http.StatusBadRequest)
		}
	})
}

// ExampleWebPageHandlerFactory returns a webpage similar to example.org's one when the domain is
// www.example.{com,org} and redirects to www.example.{com,org} when it is example.{com,org}.
func ExampleWebPageHandlerFactory() HTTPHandlerFactory {
	return HTTPHandlerFactoryFunc(func(env NetStackServerFactoryEnv, stack *netem.UNetStack) http.Handler {
		return ExampleWebPageHandler()
	})
}

// Blockpage is the webpage returned by [BlockpageHandlerFactory].
const Blockpage = `<!doctype html>
<html>
<head>
	<title>Access Denied</title>
</head>
<body>
<div>
	<h1>Access Denied</h1>
	<p>This request cannot be served in your jurisdiction.</p>
</div>
</body>
</html>
`

// BlockpageHandlerFactory returns a blockpage regardless of the incoming domain.
func BlockpageHandlerFactory() HTTPHandlerFactory {
	return HTTPHandlerFactoryFunc(func(env NetStackServerFactoryEnv, stack *netem.UNetStack) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("Date", "Thu, 24 Aug 2023 14:35:29 GMT")
			_, _ = w.Write([]byte(Blockpage))
		})
	})
}

// DefaultURLShortenerMapping is the default URL shortener mapping we use.
var DefaultURLShortenerMapping = map[string]string{
	"/21645": "https://www.example.com/",
	"/32447": "http://www.example.com/",
	"/24561": "https://example.com/",
	"/21309": "http://example.com/",
	"/30744": "https://www.example.org/",
	"/23894": "http://www.example.org/",
	"/30179": "https://example.org/",
	"/11372": "http://example.org/",
}

// URLShortenerFactory returns an [HTTPHandlerFactory] that eventually redirects
// requests using the map provided as argument or returns 404.
func URLShortenerFactory(mapping map[string]string) HTTPHandlerFactory {
	return HTTPHandlerFactoryFunc(func(env NetStackServerFactoryEnv, stack *netem.UNetStack) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("Date", "Thu, 24 Aug 2023 14:35:29 GMT")
			location, found := mapping[r.URL.Path]
			if !found {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			w.Header().Set("Location", location)
			w.WriteHeader(http.StatusPermanentRedirect)
		})
	})
}
