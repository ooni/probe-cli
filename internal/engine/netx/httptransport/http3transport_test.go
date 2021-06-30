package httptransport_test

import (
	"testing"

	"github.com/ooni/probe-cli/v3/internal/engine/netx/httptransport"
)

func TestNewHTTP3Transport(t *testing.T) {
	// mainly to cover a line which otherwise won't be directly covered
	httptransport.NewHTTP3Transport(httptransport.Config{})
}
