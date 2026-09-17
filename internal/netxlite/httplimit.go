package netxlite

import (
	"io"
	"net/http"
)

// MaxPayloadSize is 128MB
const MaxPayloadSize = int64(1 << 27)

// LimitBodyReader returns a LimitedReader capped at min(MaxPayloadSize, http.Response.ContentLength)
func LimitBodyReader(resp *http.Response) io.Reader {
	size := MaxPayloadSize
	if (0 < resp.ContentLength) && (resp.ContentLength < MaxPayloadSize) {
		size = resp.ContentLength
	}
	return io.LimitReader(resp.Body, size)
}
