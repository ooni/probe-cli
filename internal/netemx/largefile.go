package netemx

import (
	"crypto/rand"
	"net/http"

	"github.com/ooni/netem"
)

// LargeFileHandlerFactory returns an [HTTPHandlerFactory] for constructing a [LargeFileHandler].
func LargeFileHandlerFactory() HTTPHandlerFactory {
	return HTTPHandlerFactoryFunc(func(env NetStackServerFactoryEnv, stack *netem.UNetStack) http.Handler {
		return LargeFileHandler(rand.Read)
	})
}

// LargeFileHandler returns an [http.Handler] that returns a 32 MiB file.
func LargeFileHandler(reader func(b []byte) (n int, err error)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Alt-Svc", `h3=":443"`)
		w.Header().Add("Date", "Thu, 24 Aug 2023 14:35:29 GMT")
		data := make([]byte, 1<<25)
		if _, err := reader(data); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, _ = w.Write(data)
	})
}
