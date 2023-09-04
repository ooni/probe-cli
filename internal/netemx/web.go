package netemx

import (
	"net"
	"net/http"

	"github.com/ooni/netem"
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
</div>
</body>
</html>
`

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
			w.Write([]byte(ExampleWebPage))

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
	return HTTPHandlerFactoryFunc(func(_ *netem.UNetStack) http.Handler {
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

// TODO(bassosimone): it is not realistic that this webserver is able to serve valid
// blockpages over TLS but unfortunately this is currently a netem limitation.

// BlockpageHandlerFactory returns a blockpage regardless of the incoming domain.
func BlockpageHandlerFactory() HTTPHandlerFactory {
	return HTTPHandlerFactoryFunc(func(_ *netem.UNetStack) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("Alt-Svc", `h3=":443"`)
			w.Header().Add("Date", "Thu, 24 Aug 2023 14:35:29 GMT")
			w.Write([]byte(Blockpage))
		})
	})
}
