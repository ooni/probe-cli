package batch

import (
	j "encoding/json"
	"io"
	"os"
	"sync"

	"github.com/apex/log"
)

// Default handler outputting to stdout. We want to emit the batch
// output on the standard output, for two reasons:
//
// 1. because third party libraries MAY log on the stderr and
// their logs are most likely not JSON;
//
// 2. because this enables piping to `jq` or other tools in
// a much more natural way than when emitting on stderr.
//
// See https://github.com/ooni/probe/issues/1384.
var Default = New(os.Stdout)

// Handler implementation.
type Handler struct {
	*j.Encoder
	mu sync.Mutex
}

// New handler.
func New(w io.Writer) *Handler {
	return &Handler{
		Encoder: j.NewEncoder(w),
	}
}

// HandleLog implements log.Handler.
func (h *Handler) HandleLog(e *log.Entry) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.Encoder.Encode(e)
}
