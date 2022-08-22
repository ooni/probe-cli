//go:build go1.19

package tunnel

//
// Psiphon not working with go1.19: TODO(https://github.com/ooni/probe/issues/2222)
//

import (
	"context"
	"errors"
)

// psiphonStart starts the psiphon tunnel.
func psiphonStart(ctx context.Context, config *Config) (Tunnel, DebugInfo, error) {
	return nil, DebugInfo{}, errors.New(
		"psiphon is disabled when building with go1.19: see https://github.com/ooni/probe/issues/2222 for more information",
	)
}
