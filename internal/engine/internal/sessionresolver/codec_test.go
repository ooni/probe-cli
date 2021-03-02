package sessionresolver

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

type FakeCodec struct {
	EncodeData []byte
	EncodeErr  error
	DecodeErr  error
}

func (c *FakeCodec) Encode(v interface{}) ([]byte, error) {
	return c.EncodeData, c.EncodeErr
}

func (c *FakeCodec) Decode(b []byte, v interface{}) error {
	return c.DecodeErr
}

func TestCodecCustom(t *testing.T) {
	c := &FakeCodec{}
	reso := &Resolver{codec: c}
	if r := reso.getCodec(); r != c {
		t.Fatal("not the codec we expected")
	}
}

func TestCodecDefault(t *testing.T) {
	reso := &Resolver{}
	in := resolverinfo{
		URL:   "https://dns.google/dns.query",
		Score: 0.99,
	}
	data, err := reso.getCodec().Encode(in)
	if err != nil {
		t.Fatal(err)
	}
	var out resolverinfo
	if err := reso.getCodec().Decode(data, &out); err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(in, out); diff != "" {
		t.Fatal(diff)
	}
}
