package httptransport

import (
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// UserAgentTransport is a transport that ensures that we always
// set an OONI specific default User-Agent header.
type UserAgentTransport = netxlite.UserAgentTransport
