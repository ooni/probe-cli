package measurexlite

import (
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func TestNewUDPListener(t *testing.T) {
	// Make sure that we're forwarding the call to the measuring network.
	expectListener := &mocks.UDPListener{}
	trace := NewTrace(0, time.Now())
	trace.Netx = &mocks.MeasuringNetwork{
		MockNewUDPListener: func() model.UDPListener {
			return expectListener
		},
	}
	listener := trace.NewUDPListener()
	if listener != expectListener {
		t.Fatal("unexpected listener")
	}
}
