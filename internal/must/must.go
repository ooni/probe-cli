// Package must contains functions that panic on error.
package must

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"os"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/shellx"
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

// Run is like [shellx.Run] but calls [runtimex.PanicOnError] on failure.
func Run(logger model.Logger, command string, args ...string) {
	err := shellx.Run(logger, command, args...)
	runtimex.PanicOnError(err, "shellx.Run failed")
}

// RunQuiet is like [shellx.RunQuiet] but calls [runtimex.PanicOnError] on failure.
func RunQuiet(command string, args ...string) {
	err := shellx.RunQuiet(command, args...)
	runtimex.PanicOnError(err, "shellx.RunQuiet failed")
}

// RunCommandLine is like [shellx.RunCommandLine] but calls
// [runtimex.PanicOnError] on failure.
func RunCommandLine(logger model.Logger, cmdline string) {
	err := shellx.RunCommandLine(logger, cmdline)
	runtimex.PanicOnError(err, "shellx.RunCommandLine failed")
}

// RunCommandLineQuiet is like [shellx.RunCommandLineQuiet] but calls
// [runtimex.PanicOnError] on failure.
func RunCommandLineQuiet(cmdline string) {
	err := shellx.RunCommandLineQuiet(cmdline)
	runtimex.PanicOnError(err, "shellx.RunCommandLineQuiet failed")
}

// WriteFile is like [os.WriteFile] but calls
// [runtimex.PanicOnError] on failure.
func WriteFile(filename string, content []byte, mode fs.FileMode) {
	err := os.WriteFile(filename, content, mode)
	runtimex.PanicOnError(err, "os.WriteFile failed")
}

// ReadFile is like [os.ReadFile] but calls
// [runtimex.PanicOnError] on failure.
func ReadFile(filename string) []byte {
	data, err := os.ReadFile(filename)
	runtimex.PanicOnError(err, "os.ReadFile failed")
	return data
}

// FirstLineBytes takes in input a sequence of bytes and
// returns in output the first line. This function will
// call [runtimex.PanicOnError] on failure.
func FirstLineBytes(data []byte) []byte {
	first, _, good := bytes.Cut(data, []byte("\n"))
	runtimex.Assert(good, "could not find the first line")
	return first
}
