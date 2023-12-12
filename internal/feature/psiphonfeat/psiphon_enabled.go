//go:build !go1.21 && !ooni_feature_disable_psiphon

package psiphonfeat

import (
	"context"

	"github.com/Psiphon-Labs/psiphon-tunnel-core/ClientLibrary/clientlib"
)

// Start attempts to start the Psiphon tunnel and returns either a Tunnel or an error.
func Start(ctx context.Context, config []byte, workdir string) (Tunnel, error) {
	return clientlib.StartTunnel(ctx, config, "", clientlib.Parameters{
		DataRootDirectory: &workdir}, nil, nil)
}
