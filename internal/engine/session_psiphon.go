// +build ooni_psiphon_config

package engine

import (
	"bytes"
	"context"
	_ "embed"
	"io/ioutil"

	"filippo.io/age"
)

//go:embed psiphon-config.json.age
var psiphonConfigJSONAge []byte

//go:embed psiphon-config.key
var psiphonConfigSecretKey string

// FetchPsiphonConfig decrypts psiphonConfigJSONAge using
// filippo.io/age _and_ psiphonConfigSecretKey.
func (s *Session) FetchPsiphonConfig(ctx context.Context) ([]byte, error) {
	key := "AGE-SECRET-KEY-1" + psiphonConfigSecretKey
	identity, err := age.ParseX25519Identity(key)
	if err != nil {
		return nil, err
	}
	input := bytes.NewReader(psiphonConfigJSONAge)
	output, err := age.Decrypt(input, identity)
	if err != nil {
		return nil, err
	}
	return ioutil.ReadAll(output)
}
