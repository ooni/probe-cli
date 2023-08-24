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

// ExampleWebPageHandlerFactory returns a webpage similar to example.org's one.
func ExampleWebPageHandlerFactory() QAEnvHTTPHandlerFactory {
	return QAEnvHTTPHandlerFactoryFunc(func(_ netem.UnderlyingNetwork) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("Alt-Svc", `h3=":443"`)
			w.Header().Add("Date", "Thu, 24 Aug 2023 14:35:29 GMT")
			w.Write([]byte(ExampleWebPage))
		})
	})
}
