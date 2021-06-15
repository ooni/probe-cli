package ooapi

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/ooni/probe-cli/v3/internal/iox"
)

type FakeCodec struct {
	DecodeErr  error
	EncodeData []byte
	EncodeErr  error
}

func (mc *FakeCodec) Encode(v interface{}) ([]byte, error) {
	return mc.EncodeData, mc.EncodeErr
}

func (mc *FakeCodec) Decode(b []byte, v interface{}) error {
	return mc.DecodeErr
}

type FakeHTTPClient struct {
	Err  error
	Resp *http.Response
}

func (c *FakeHTTPClient) Do(req *http.Request) (*http.Response, error) {
	time.Sleep(10 * time.Microsecond)
	if req.Body != nil {
		_, _ = iox.ReadAllContext(req.Context(), req.Body)
		req.Body.Close()
	}
	if c.Err != nil {
		return nil, c.Err
	}
	c.Resp.Request = req // non thread safe but it doesn't matter
	return c.Resp, nil
}

type FakeBody struct {
	Data []byte
	Err  error
}

func (fb *FakeBody) Read(p []byte) (int, error) {
	time.Sleep(10 * time.Microsecond)
	if fb.Err != nil {
		return 0, fb.Err
	}
	if len(fb.Data) <= 0 {
		return 0, io.EOF
	}
	n := copy(p, fb.Data)
	fb.Data = fb.Data[n:]
	return n, nil
}

func (fb *FakeBody) Close() error {
	return nil
}

type FakeRequestMaker struct {
	Req *http.Request
	Err error
}

func (frm *FakeRequestMaker) NewRequest(
	ctx context.Context, method, URL string, body io.Reader) (*http.Request, error) {
	return frm.Req, frm.Err
}

type FakeTemplateExecutor struct {
	Out string
	Err error
}

func (fte *FakeTemplateExecutor) Execute(tmpl string, v interface{}) (string, error) {
	return fte.Out, fte.Err
}

type FakeKVStore struct {
	SetError error
	GetData  []byte
	GetError error
}

func (fs *FakeKVStore) Get(key string) ([]byte, error) {
	return fs.GetData, fs.GetError
}

func (fs *FakeKVStore) Set(key string, value []byte) error {
	return fs.SetError
}
