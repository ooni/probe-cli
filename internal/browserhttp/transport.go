package browserhttp

//
// An HTTP transport that writes requests the way Chrome does
//
// net/http writes headers in its own order and case, and its HTTP/2
// SETTINGS, window update and pseudo-header order differ from Chrome's.
//

import (
	"bufio"
	"bytes"
	"compress/flate"
	"compress/gzip"
	"compress/zlib"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/andybalholm/brotli"
	"github.com/klauspost/compress/zstd"
	"github.com/ooni/probe-cli/v3/internal/model"
	"golang.org/x/net/http/httpguts"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/hpack"
)

// Chrome's HTTP/2 connection settings, in the order it sends them.
var h2Settings = []http2.Setting{
	{ID: http2.SettingHeaderTableSize, Val: 65536},
	{ID: http2.SettingEnablePush, Val: 0},
	{ID: http2.SettingInitialWindowSize, Val: 6291456},
	{ID: http2.SettingMaxHeaderListSize, Val: 262144},
}

// h2WindowUpdate grows the connection window to Chrome's 15 MiB.
const h2WindowUpdate = 15663105

// H2Priority is the priority Chrome gives a navigation's HEADERS:
// weight 256 (sent as 255), exclusive, on the root.
var H2Priority = http2.PriorityParam{StreamDep: 0, Exclusive: true, Weight: 255}

// Transport sends one request over an established connection, as
// HTTP/2 when ALPN negotiated h2 and as HTTP/1.1 otherwise.
type Transport struct {
	alpn string
	conn net.Conn
}

var _ model.HTTPTransport = &Transport{}

// NewTransport returns a transport for one request over conn. Wrap
// it with [netxlite.WrapHTTPTransport] for logging and error wrapping.
func NewTransport(conn net.Conn, alpn string) *Transport {
	return &Transport{alpn: alpn, conn: conn}
}

// Network implements model.HTTPTransport.
func (txp *Transport) Network() string {
	return "tcp"
}

// CloseIdleConnections implements model.HTTPTransport. The caller owns the connection.
func (txp *Transport) CloseIdleConnections() {}

// RoundTrip implements model.HTTPTransport.
func (txp *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	// unblock reads and writes when the request's context is done
	stop := context.AfterFunc(req.Context(), func() {
		_ = txp.conn.SetDeadline(time.Unix(1, 0))
	})
	var (
		resp *http.Response
		err  error
	)
	if txp.alpn == "h2" {
		resp, err = txp.roundTripH2(req)
	} else {
		resp, err = txp.roundTripH1(req)
	}
	if err != nil {
		stop()
		if ctxErr := req.Context().Err(); ctxErr != nil {
			return nil, ctxErr
		}
		return nil, err
	}
	resp.Body = &stopBody{ReadCloser: resp.Body, stop: stop}
	DecodeBody(resp)
	return resp, nil
}

// stopBody stops watching the request's context once closed.
type stopBody struct {
	io.ReadCloser
	stop func() bool
}

func (b *stopBody) Close() error {
	b.stop()
	return b.ReadCloser.Close()
}

// headerField is a header as written on the wire.
type headerField struct {
	name, value string
}

// orderHeaders returns the request headers in Chrome's order, with
// names written as in order. Headers Chrome doesn't send go last, sorted.
func orderHeaders(header http.Header, order []string) (out []headerField, err error) {
	seen := map[string]bool{"Host": true}
	add := func(name string, values []string) error {
		for _, value := range values {
			if !httpguts.ValidHeaderFieldName(name) || !httpguts.ValidHeaderFieldValue(value) {
				return fmt.Errorf("invalid header field %q", name)
			}
			out = append(out, headerField{name, value})
		}
		return nil
	}
	for _, name := range order {
		key := http.CanonicalHeaderKey(name)
		seen[key] = true
		if err := add(name, header.Values(key)); err != nil {
			return nil, err
		}
	}
	var rest []string
	for key := range header {
		if !seen[http.CanonicalHeaderKey(key)] {
			rest = append(rest, key)
		}
	}
	slices.Sort(rest)
	for _, key := range rest {
		if err := add(key, header[key]); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// requestHost returns the value of Host or :authority.
func requestHost(req *http.Request) string {
	if req.Host != "" {
		return req.Host
	}
	return req.URL.Host
}

func (txp *Transport) roundTripH1(req *http.Request) (*http.Response, error) {
	fields, err := orderHeaders(req.Header, HeaderOrder)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "%s %s HTTP/1.1\r\n", req.Method, req.URL.RequestURI())
	fmt.Fprintf(&buf, "Host: %s\r\n", requestHost(req))
	buf.WriteString("Connection: keep-alive\r\n")
	for _, f := range fields {
		fmt.Fprintf(&buf, "%s: %s\r\n", f.name, f.value)
	}
	buf.WriteString("\r\n")
	if _, err := txp.conn.Write(buf.Bytes()); err != nil {
		return nil, err
	}
	return http.ReadResponse(bufio.NewReader(txp.conn), req)
}

// errH2 is the error for a failed HTTP/2 exchange.
var errH2 = errors.New("http2")

func (txp *Transport) roundTripH2(req *http.Request) (*http.Response, error) {
	order := []string{}
	for _, name := range HeaderOrder {
		order = append(order, strings.ToLower(name))
	}
	fields, err := orderHeaders(req.Header, order)
	if err != nil {
		return nil, err
	}

	// Chrome's pseudo-header order is m,a,s,p and priority comes last
	var block bytes.Buffer
	enc := hpack.NewEncoder(&block)
	for _, f := range []hpack.HeaderField{
		{Name: ":method", Value: req.Method},
		{Name: ":authority", Value: requestHost(req)},
		{Name: ":scheme", Value: "https"},
		{Name: ":path", Value: req.URL.RequestURI()},
	} {
		_ = enc.WriteField(f)
	}
	for _, f := range fields {
		_ = enc.WriteField(hpack.HeaderField{Name: strings.ToLower(f.name), Value: f.value})
	}
	_ = enc.WriteField(hpack.HeaderField{Name: "priority", Value: "u=0, i"})

	// write the preface, SETTINGS, WINDOW_UPDATE and HEADERS together
	w := bufio.NewWriter(txp.conn)
	if _, err := w.WriteString(http2.ClientPreface); err != nil {
		return nil, err
	}
	fr := http2.NewFramer(w, txp.conn)
	fr.ReadMetaHeaders = hpack.NewDecoder(h2Settings[0].Val, nil)
	fr.MaxHeaderListSize = h2Settings[3].Val
	if err := fr.WriteSettings(h2Settings...); err != nil {
		return nil, err
	}
	if err := fr.WriteWindowUpdate(0, h2WindowUpdate); err != nil {
		return nil, err
	}
	if err := fr.WriteHeaders(http2.HeadersFrameParam{
		StreamID:      1,
		BlockFragment: block.Bytes(),
		EndStream:     true,
		EndHeaders:    true,
		Priority:      H2Priority,
	}); err != nil {
		return nil, err
	}
	if err := w.Flush(); err != nil {
		return nil, err
	}

	sc := &h2Stream{fr: fr, w: w}
	for {
		frame, err := sc.next()
		if err != nil {
			return nil, err
		}
		hf, ok := frame.(*http2.MetaHeadersFrame)
		if !ok {
			return nil, fmt.Errorf("%w: DATA before HEADERS", errH2)
		}
		resp, err := sc.response(req, hf)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode >= 100 && resp.StatusCode < 200 {
			continue // e.g. 103 Early Hints
		}
		return resp, nil
	}
}

// h2Stream reads the frames of stream 1, our request's.
type h2Stream struct {
	fr      *http2.Framer
	w       *bufio.Writer
	mu      sync.Mutex
	unacked uint32
}

// next returns the next HEADERS or DATA frame of stream 1, answering
// SETTINGS and PING on the way.
func (sc *h2Stream) next() (http2.Frame, error) {
	for {
		frame, err := sc.fr.ReadFrame()
		if err != nil {
			return nil, err
		}
		switch f := frame.(type) {
		case *http2.SettingsFrame:
			if !f.IsAck() {
				if err := sc.write(func() error { return sc.fr.WriteSettingsAck() }); err != nil {
					return nil, err
				}
			}
		case *http2.PingFrame:
			if !f.IsAck() {
				if err := sc.write(func() error { return sc.fr.WritePing(true, f.Data) }); err != nil {
					return nil, err
				}
			}
		case *http2.GoAwayFrame:
			if f.LastStreamID < 1 {
				return nil, fmt.Errorf("%w: GOAWAY %s", errH2, f.ErrCode)
			}
		case *http2.RSTStreamFrame:
			if f.StreamID == 1 {
				return nil, fmt.Errorf("%w: RST_STREAM %s", errH2, f.ErrCode)
			}
		case *http2.MetaHeadersFrame:
			if f.StreamID == 1 {
				return f, nil
			}
		case *http2.DataFrame:
			if f.StreamID == 1 {
				if err := sc.consumed(f.Length); err != nil {
					return nil, err
				}
				return f, nil
			}
		}
	}
}

// consumed returns flow control credit once we have read 1 MiB, well
// within Chrome's 6 MiB stream window.
func (sc *h2Stream) consumed(n uint32) error {
	sc.unacked += n
	if sc.unacked < 1<<20 {
		return nil
	}
	inc := sc.unacked
	sc.unacked = 0
	return sc.write(func() error {
		if err := sc.fr.WriteWindowUpdate(0, inc); err != nil {
			return err
		}
		return sc.fr.WriteWindowUpdate(1, inc)
	})
}

func (sc *h2Stream) write(fn func() error) error {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	if err := fn(); err != nil {
		return err
	}
	return sc.w.Flush()
}

// response converts a HEADERS frame into a response whose body reads
// the stream's DATA frames.
func (sc *h2Stream) response(req *http.Request, hf *http2.MetaHeadersFrame) (*http.Response, error) {
	status, err := strconv.Atoi(hf.PseudoValue("status"))
	if err != nil {
		return nil, fmt.Errorf("%w: bad :status", errH2)
	}
	header := http.Header{}
	for _, f := range hf.RegularFields() {
		header.Add(http.CanonicalHeaderKey(f.Name), f.Value)
	}
	contentLength := int64(-1)
	if cl := header.Get("Content-Length"); cl != "" {
		if v, err := strconv.ParseInt(cl, 10, 64); err == nil && v >= 0 {
			contentLength = v
		}
	}
	return &http.Response{
		Status:        fmt.Sprintf("%d %s", status, http.StatusText(status)),
		StatusCode:    status,
		Proto:         "HTTP/2.0",
		ProtoMajor:    2,
		Header:        header,
		Body:          &h2Body{sc: sc, done: hf.StreamEnded()},
		ContentLength: contentLength,
		Request:       req,
	}, nil
}

// h2Body reads the DATA frames of stream 1.
type h2Body struct {
	sc   *h2Stream
	buf  []byte
	done bool
	err  error
}

func (b *h2Body) Read(p []byte) (int, error) {
	for len(b.buf) == 0 {
		if b.done {
			return 0, io.EOF
		}
		if b.err != nil {
			return 0, b.err
		}
		frame, err := b.sc.next()
		if err != nil {
			b.err = err
			continue
		}
		switch f := frame.(type) {
		case *http2.DataFrame:
			b.buf = append(b.buf[:0], f.Data()...)
			b.done = f.StreamEnded()
		case *http2.MetaHeadersFrame:
			b.done = f.StreamEnded() // trailers
		}
	}
	n := copy(p, b.buf)
	b.buf = b.buf[n:]
	return n, nil
}

func (b *h2Body) Close() error {
	return nil
}

// DecodeBody decodes the body we asked to be compressed. The headers
// stay as received; the body is what a browser would render. The caller's
// read limit applies to the decoded bytes.
func DecodeBody(resp *http.Response) {
	var open func(r io.Reader) (io.Reader, func(), error)
	switch strings.ToLower(strings.TrimSpace(resp.Header.Get("Content-Encoding"))) {
	case "gzip", "x-gzip":
		open = func(r io.Reader) (io.Reader, func(), error) {
			zr, err := gzip.NewReader(r)
			return zr, func() {}, err
		}
	case "deflate":
		open = func(r io.Reader) (io.Reader, func(), error) {
			// zlib-wrapped, as the standard says, or raw, as some servers send
			br := bufio.NewReader(r)
			if head, err := br.Peek(2); err == nil && head[0]&0x0f == 8 && (uint16(head[0])<<8|uint16(head[1]))%31 == 0 {
				zr, err := zlib.NewReader(br)
				return zr, func() {}, err
			}
			return flate.NewReader(br), func() {}, nil
		}
	case "br":
		open = func(r io.Reader) (io.Reader, func(), error) {
			return brotli.NewReader(r), func() {}, nil
		}
	case "zstd":
		open = func(r io.Reader) (io.Reader, func(), error) {
			zr, err := zstd.NewReader(r, zstd.WithDecoderLowmem(true), zstd.WithDecoderMaxWindow(8<<20))
			if err != nil {
				return nil, nil, err
			}
			return zr, zr.Close, nil
		}
	default:
		return
	}
	resp.Body = &decodedBody{body: resp.Body, open: open}
	resp.Uncompressed = true
}

// decodedBody starts decoding on the first read, so that an empty
// body (e.g. a redirect) isn't an error.
type decodedBody struct {
	body   io.ReadCloser
	open   func(r io.Reader) (io.Reader, func(), error)
	dec    io.Reader
	closer func()
	err    error
}

func (b *decodedBody) Read(p []byte) (int, error) {
	if b.dec == nil && b.err == nil {
		br := bufio.NewReader(b.body)
		if _, err := br.Peek(1); err != nil {
			b.err = err // io.EOF for an empty body
		} else {
			b.dec, b.closer, b.err = b.open(br)
		}
	}
	if b.err != nil {
		return 0, b.err
	}
	return b.dec.Read(p)
}

func (b *decodedBody) Close() error {
	if b.closer != nil {
		b.closer()
	}
	return b.body.Close()
}
