//go:build !go1.24 && !ooni_feature_disable_psiphon

package psiphonfeat

import (
	"context"

	"github.com/Psiphon-Labs/psiphon-tunnel-core/ClientLibrary/clientlib"
)

// Enabled indicates whether this feature is enabled.
const Enabled = true

// Start attempts to start the Psiphon tunnel and returns either a Tunnel or an error.
func Start(ctx context.Context, config []byte, workdir string) (Tunnel, error) {
	tun, err := clientlib.StartTunnel(ctx, config, "", clientlib.Parameters{
		DataRootDirectory: &workdir}, nil, nil)
	if err != nil {
		return nil, err
	}
	return &tunnel{tun}, nil
}

type tunnel struct {
	*clientlib.PsiphonTunnel
}

// GetSOCKSProxyPort implements Tunnel.
func (t *tunnel) GetSOCKSProxyPort() int {
	return t.SOCKSProxyPort
}
