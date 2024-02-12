package netemx

import (
	"net"
	"net/http"

	"github.com/ooni/netem"
)

// HTTPBinHandlerFactory constructs an [HTTPBinHandler].
func HTTPBinHandlerFactory() HTTPHandlerFactory {
	return HTTPHandlerFactoryFunc(func(env NetStackServerFactoryEnv, stack *netem.UNetStack) http.Handler {
		return HTTPBinHandler()
	})
}

// HTTPBinHandler returns the [http.Handler] implementing an httpbin.com-like service.
//
// We currently implement the following API endpoints:
//
//	/broken-redirect-http
//		When accessed by the OONI Probe client redirects with 302 to http:// and
//		otherwise redirects to the https://www.example.com/ URL.
//
//	/broken-redirect-https
//		When accessed by the OONI Probe client redirects with 302 to https:// and
//		otherwise redirects to the https://www.example.com/ URL.
//
// Any other request URL causes a 404 respose.
func HTTPBinHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Date", "Thu, 24 Aug 2023 14:35:29 GMT")

		// missing address => 500
		address, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// compute variables used by the switch below
		cleartextRedirect := r.URL.Path == "/broken-redirect-http"
		client := address == DefaultClientAddress
		secureRedirect := r.URL.Path == "/broken-redirect-https"

		switch {
		// broken HTTP redirect for clients
		case cleartextRedirect && client:
			w.Header().Set("Location", "http://")
			w.WriteHeader(http.StatusFound)

		// working HTTP redirect for anyone else
		case cleartextRedirect && !client:
			w.Header().Set("Location", "http://www.example.com/")
			w.WriteHeader(http.StatusFound)

		// broken HTTPS redirect for clients
		case secureRedirect && client:
			w.Header().Set("Location", "https://")
			w.WriteHeader(http.StatusFound)

		// working HTTPS redirect for anyone else
		case secureRedirect && !client:
			w.Header().Set("Location", "https://www.example.com/")
			w.WriteHeader(http.StatusFound)

		// otherwise
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
}
