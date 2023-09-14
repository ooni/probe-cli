package engineresolver

//
// JSON codec
//

import "encoding/json"

// jsonCodec encodes to/decodes from JSON.
type jsonCodec interface {
	// Encode encodes v as a JSON stream of bytes.
	Encode(v interface{}) ([]byte, error)

	// Decode decodes b from a JSON stream of bytes.
	Decode(b []byte, v interface{}) error
}

// codec always returns a valid jsonCodec.
func (r *Resolver) codec() jsonCodec {
	if r.jsonCodec != nil {
		return r.jsonCodec
	}
	return &jsonCodecStdlib{}
}

// jsonCodecStdlib is the default codec.
type jsonCodecStdlib struct{}

// Decode implements jsonCodec.Decode.
func (*jsonCodecStdlib) Decode(b []byte, v interface{}) error {
	return json.Unmarshal(b, v)
}

// Encode implements jsonCodec.Encode.
func (*jsonCodecStdlib) Encode(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}
