package oocryptofeat

import (
	oohttp "github.com/ooni/oohttp"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// Make sure the two models are compatible with each other.
var _ model.TLSConn = oohttp.TLSConn(nil)
