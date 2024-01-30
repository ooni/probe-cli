package netemx

import (
	"log"
	"net"
	"net/http"

	"github.com/ooni/netem"
)

// HTTPBinHandlerFactory implements httpbin.com.
func HTTPBinHandlerFactory() HTTPHandlerFactory {
	return HTTPHandlerFactoryFunc(func(env NetStackServerFactoryEnv, stack *netem.UNetStack) http.Handler {
		return HTTPBinHandler()
	})
}

// HTTPBinHandler returns the [http.Handler] for httpbin.
func HTTPBinHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// missing address => 500
		address, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			log.Printf("CLOUDFLARE_CACHE: missing address in request => 500")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if r.URL.Path == "/broken-redirect-http" {
			log.Printf("ELLIOT: %+v", r.URL)
			// See https://github.com/ooni/probe/issues/2628
			if address == DefaultClientAddress {
				w.Header().Set("Location", "http://")
			} else {
				w.Header().Set("Location", "http://www.example.com/")
			}
			w.WriteHeader(http.StatusFound)
			return
		}

		if r.URL.Path == "/broken-redirect-https" {
			log.Printf("ELLIOT: %+v", r.URL)
			// See https://github.com/ooni/probe/issues/2628
			if address == DefaultClientAddress {
				w.Header().Set("Location", "https://")
			} else {
				w.Header().Set("Location", "https://www.example.com/")
			}
			w.WriteHeader(http.StatusFound)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	})
}
