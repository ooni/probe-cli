package netemx

import (
	"net"
	"net/http"

	"github.com/ooni/netem"
)

// YandexHandlerFactory implements yandex.com.
func YandexHandlerFactory() HTTPHandlerFactory {
	return HTTPHandlerFactoryFunc(func(env NetStackServerFactoryEnv, stack *netem.UNetStack) http.Handler {
		return YandexHandler()
	})
}

// YandexHandler returns the [http.Handler] for yandex.com.
func YandexHandler() http.Handler {
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
		case "ya.ru":
			_, _ = w.Write([]byte(ExampleWebPage))

		case "yandex.com":
			w.Header().Add("Location", "https://ya.ru/")
			w.WriteHeader(http.StatusPermanentRedirect)

		case "xn--d1acpjx3f.xn--p1ai":
			w.Header().Add("Location", "https://yandex.com/")
			w.WriteHeader(http.StatusPermanentRedirect)

		default:
			w.WriteHeader(http.StatusBadRequest)
		}
	})
}
