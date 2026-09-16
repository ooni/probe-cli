//go:build !(aix || darwin || dragonfly || freebsd || (js && wasm) || linux || nacl || netbsd || openbsd || solaris)

package tlsmiddlebox

import (
	"sync"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
)

func tracerouteTCP(_ int64, _ time.Time, _ string, _ int, _ int, wg *sync.WaitGroup, _ model.Logger, _ string) (*ICMPIteration, error) {
	defer wg.Done()

	return nil, nil
}
