package enginenetx

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// HTTPSDialerLoadablePolicy is an [HTTPSDialerPolicy] that you
// can load from its JSON serialization on disk.
type HTTPSDialerLoadablePolicy struct {
	// Domains maps each domain to its policy. When there is
	// no domain, the code falls back to the default "null" policy
	// implemented by HTTPSDialerNullPolicy.
	Domains map[string][]*HTTPSDialerLoadableTactic
}

// LoadHTTPSDialerPolicy loads the [HTTPSDialerPolicy] from
// the given bytes containing a serialized JSON object.
func LoadHTTPSDialerPolicy(data []byte) (*HTTPSDialerLoadablePolicy, error) {
	var p HTTPSDialerLoadablePolicy
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// HTTPSDialerLoadableTactic is an [HTTPSDialerTactic] that you
// can load from JSON as part of [HTTPSDialerLoadablePolicy].
type HTTPSDialerLoadableTactic struct {
	// IPAddr is the IP address to use for dialing.
	IPAddr string

	// InitialDelay is the time in nanoseconds after which
	// you would like to start this policy.
	InitialDelay time.Duration

	// SNI is the TLS ServerName to send over the wire.
	SNI string

	// VerifyHostname is the hostname using during
	// the X.509 certificate verification.
	VerifyHostname string
}

// HTTPSDialerLoadableTacticWrapper wraps [HTTPSDialerLoadableTactic]
// to make it implements the [HTTPSDialerTactic] interface.
type HTTPSDialerLoadableTacticWrapper struct {
	Tactic *HTTPSDialerLoadableTactic
}

// IPAddr implements HTTPSDialerTactic.
func (ldt *HTTPSDialerLoadableTacticWrapper) IPAddr() string {
	return ldt.Tactic.IPAddr
}

// InitialDelay implements HTTPSDialerTactic.
func (ldt *HTTPSDialerLoadableTacticWrapper) InitialDelay() time.Duration {
	return ldt.Tactic.InitialDelay
}

// NewTLSHandshaker implements HTTPSDialerTactic.
func (ldt *HTTPSDialerLoadableTacticWrapper) NewTLSHandshaker(netx *netxlite.Netx, logger model.Logger) model.TLSHandshaker {
	return netxlite.NewTLSHandshakerStdlib(logger)
}

// OnStarting implements HTTPSDialerTactic.
func (ldt *HTTPSDialerLoadableTacticWrapper) OnStarting() {
	// TODO(bassosimone): implement
}

// OnSuccess implements HTTPSDialerTactic.
func (ldt *HTTPSDialerLoadableTacticWrapper) OnSuccess() {
	// TODO(bassosimone): implement
}

// OnTCPConnectError implements HTTPSDialerTactic.
func (ldt *HTTPSDialerLoadableTacticWrapper) OnTCPConnectError(ctx context.Context, err error) {
	// TODO(bassosimone): implement
}

// OnTLSHandshakeError implements HTTPSDialerTactic.
func (ldt *HTTPSDialerLoadableTacticWrapper) OnTLSHandshakeError(ctx context.Context, err error) {
	// TODO(bassosimone): implement
}

// OnTLSVerifyError implements HTTPSDialerTactic.
func (ldt *HTTPSDialerLoadableTacticWrapper) OnTLSVerifyError(ctz context.Context, err error) {
	// TODO(bassosimone): implement
}

// SNI implements HTTPSDialerTactic.
func (ldt *HTTPSDialerLoadableTacticWrapper) SNI() string {
	return ldt.Tactic.SNI
}

// String implements HTTPSDialerTactic.
func (ldt *HTTPSDialerLoadableTacticWrapper) String() string {
	return fmt.Sprintf("%+v", ldt.Tactic)
}

// VerifyHostname implements HTTPSDialerTactic.
func (ldt *HTTPSDialerLoadableTacticWrapper) VerifyHostname() string {
	return ldt.Tactic.VerifyHostname
}
