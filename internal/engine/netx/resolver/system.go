package resolver

import "github.com/ooni/probe-cli/v3/internal/engine/netx/selfcensor"

// SystemResolver is the system resolver. It is implemented using
// selfcensor.SystemResolver so that we can perform integration testing
// by forcing the code to return specific responses.
type SystemResolver = selfcensor.SystemResolver

var _ Resolver = SystemResolver{}
