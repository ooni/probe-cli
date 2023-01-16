package sessionresolver

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

type jsonCodecMockable struct {
	EncodeData []byte
	EncodeErr  error
	DecodeErr  error
}

func (c *jsonCodecMockable) Encode(v interface{}) ([]byte, error) {
	return c.EncodeData, c.EncodeErr
}

func (c *jsonCodecMockable) Decode(b []byte, v interface{}) error {
	return c.DecodeErr
}

func TestJSONCodecCustom(t *testing.T) {
	c := &jsonCodecMockable{}
	reso := &Resolver{jsonCodec: c}
	if r := reso.codec(); r != c {
		t.Fatal("not the codec we expected")
	}
}

func TestJSONCodecDefault(t *testing.T) {
	reso := &Resolver{}
	in := resolverinfo{
		URL:   "https://dns.google/dns.query",
		Score: 0.99,
	}
	data, err := reso.codec().Encode(in)
	if err != nil {
		t.Fatal(err)
	}
	var out resolverinfo
	if err := reso.codec().Decode(data, &out); err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(in, out); diff != "" {
		t.Fatal(diff)
	}
}
