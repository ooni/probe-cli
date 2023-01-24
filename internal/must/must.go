// Package must contains functions that panic on error.
package must

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"

	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// CreateFile is like [os.Create] but calls
// [runtimex.PanicOnError] on failure.
func CreateFile(name string) *File {
	fp, err := os.Create(name)
	runtimex.PanicOnError(err, "os.Create failed")
	return &File{fp}
}

// OpenFile is like [os.Open] but calls
// [runtimex.PanicOnError] on failure.
func OpenFile(name string) *File {
	fp, err := os.Open(name)
	runtimex.PanicOnError(err, "os.Open failed")
	return &File{fp}
}

// File wraps [os.File].
type File struct {
	*os.File
}

// MustClose is like [os.File.Close] but calls
// [runtimex.PanicOnError] on failure.
func (fp *File) MustClose() {
	err := fp.File.Close()
	runtimex.PanicOnError(err, "fp.File.Close failed")
}

// Fprintf is like [fmt.Fprintf] but calls
// [runtimex.PanicOnError] on failure.
func Fprintf(w io.Writer, format string, v ...any) {
	_, err := fmt.Fprintf(w, format, v...)
	runtimex.PanicOnError(err, "fmt.Fprintf failed")
}

// ParseURL is like [url.Parse] but calls
// [runtimex.PanicOnError] on failure.
func ParseURL(URL string) *url.URL {
	parsed, err := url.Parse(URL)
	runtimex.PanicOnError(err, "url.Parse failed")
	return parsed
}

// MarshalJSON is like [json.Marshal] but calls
// [runtimex.PanicOnError] on failure.
func MarshalJSON(v any) []byte {
	data, err := json.Marshal(v)
	runtimex.PanicOnError(err, "json.Marshal failed")
	return data
}

// MarshalAndIndentJSON is like [json.MarshalIndent] but calls
// [runtimex.PanicOnError] on failure.
func MarshalAndIndentJSON(v any, prefix string, indent string) []byte {
	data, err := json.MarshalIndent(v, prefix, indent)
	runtimex.PanicOnError(err, "json.MarshalIndent failed")
	return data
}

// UnmarshalJSON is like [json.Marshal] but calls
// [runtimex.PanicOnError] on failure.
func UnmarshalJSON(data []byte, v any) {
	err := json.Unmarshal(data, v)
	runtimex.PanicOnError(err, "json.Unmarshal failed")
}

// Listen is like [net.Listen] but calls
// [runtimex.PanicOnError] on failure.
func Listen(network string, address string) net.Listener {
	listener, err := net.Listen(network, address)
	runtimex.PanicOnError(err, "net.Listen failed")
	return listener
}

// NewHTTPRequest is like [http.NewRequest] but calls
// [runtimex.PanicOnError] on failure.
func NewHTTPRequest(method string, url string, body io.Reader) *http.Request {
	req, err := http.NewRequest(method, url, body)
	runtimex.PanicOnError(err, "http.NewRequest failed")
	return req
}

// SplitHostPort is like [net.SplitHostPort] but calls
// [runtimex.PanicOnError] on failure.
func SplitHostPort(hostport string) (host string, port string) {
	host, port, err := net.SplitHostPort(hostport)
	runtimex.PanicOnError(err, "net.SplitHostPort failed")
	return host, port
}
