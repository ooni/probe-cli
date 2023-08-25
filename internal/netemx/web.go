package netemx

import (
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

// ExampleWebPageHandlerFactory returns a webpage similar to example.org's one when the domain is
// www.example.{com,org} and redirects to www.example.{com,org} when it is example.{com,org}.
func ExampleWebPageHandlerFactory() QAEnvHTTPHandlerFactory {
	return QAEnvHTTPHandlerFactoryFunc(func(_ netem.UnderlyingNetwork) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("Alt-Svc", `h3=":443"`)
			w.Header().Add("Date", "Thu, 24 Aug 2023 14:35:29 GMT")

			switch r.Host {
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

// BlockpageHandlerFactory returns a webpage similar to example.org's one.
func BlockpageHandlerFactory() QAEnvHTTPHandlerFactory {
	return QAEnvHTTPHandlerFactoryFunc(func(_ netem.UnderlyingNetwork) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("Alt-Svc", `h3=":443"`)
			w.Header().Add("Date", "Thu, 24 Aug 2023 14:35:29 GMT")
			w.Write([]byte(Blockpage))
		})
	})
}
