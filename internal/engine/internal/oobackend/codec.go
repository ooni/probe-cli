package oobackend

import "encoding/json"

// codec is the codec we use.
type codec interface {
	// Encode encodes v as a stream of bytes.
	Encode(v interface{}) ([]byte, error)

	// Decode decodes b into a stream of bytes.
	Decode(b []byte, v interface{}) error
}

// jsonCodec always returns a valid codec.
func (c *Client) jsonCodec() codec {
	if c.codec != nil {
		return c.codec
	}
	return &defaultCodec{}
}

// defaultCodec is the default codec (JSON).
type defaultCodec struct{}

// Decode decodes b into v using the default codec.
func (*defaultCodec) Decode(b []byte, v interface{}) error {
	return json.Unmarshal(b, v)
}

// Encode encodes v using the default codec.
func (*defaultCodec) Encode(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}
