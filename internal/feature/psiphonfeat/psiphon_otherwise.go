//go:build go1.22 || ooni_feature_disable_psiphon

package psiphonfeat

import "context"

// Enabled indicates whether this feature is enabled.
const Enabled = false

// Start attempts to start the Psiphon tunnel and returns either a Tunnel or an error.
func Start(ctx context.Context, config []byte, workdir string) (Tunnel, error) {
	return nil, ErrFeatureNotEnabled
}
