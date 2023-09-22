// Package hujsonx contains github.com/tailscale/hujson extensions.
package hujsonx

import (
	"encoding/json"

	"github.com/tailscale/hujson"
)

// Unmarshal is like [json.Unmarshal] except that it first removes comments and
// extra commas using the [hujson.Standardize] function.
func Unmarshal(data []byte, v any) error {
	data, err := hujson.Standardize(data)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}
