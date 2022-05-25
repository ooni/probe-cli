//go:build ooni_psiphon_config

package engine

import (
	"bytes"
	"context"
	_ "embed"

	"filippo.io/age"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

//go:embed psiphon-config.json.age
var psiphonConfigJSONAge []byte

//go:embed psiphon-config.key
var psiphonConfigSecretKey string

// sessionTunnelEarlySession is the early session that we pass
// to tunnel.Start to fetch the Psiphon configuration.
type sessionTunnelEarlySession struct{}

// FetchPsiphonConfig decrypts psiphonConfigJSONAge using
// filippo.io/age _and_ psiphonConfigSecretKey.
func (s *sessionTunnelEarlySession) FetchPsiphonConfig(ctx context.Context) ([]byte, error) {
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
	return netxlite.ReadAllContext(ctx, output)
}

// FetchPsiphonConfig decrypts psiphonConfigJSONAge using
// filippo.io/age _and_ psiphonConfigSecretKey.
func (s *Session) FetchPsiphonConfig(ctx context.Context) ([]byte, error) {
	child := &sessionTunnelEarlySession{}
	return child.FetchPsiphonConfig(ctx)
}

// CheckEmbeddedPsiphonConfig checks whether we can load psiphon's config
func CheckEmbeddedPsiphonConfig() error {
	child := &sessionTunnelEarlySession{}
	_, err := child.FetchPsiphonConfig(context.Background())
	return err
}
