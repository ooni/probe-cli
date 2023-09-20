package enginenetx

import (
	"context"
	"fmt"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// circoTactic implements [HTTPSDialerTactic] for [CircoPolicy].
type circoTactic struct {
	// Address is the IP address to connect to.
	Address string

	// InitialWaitTime is the time to wait before starting this tactic, or
	// zero if there's no need to wait.
	InitialWaitTime time.Duration

	// TLSServerName is the SNI to send as part of the TLS Client Hello.
	TLSServerName string

	// X509VerifyHostname is the host name to use during certificate verification, which
	// will be different from the sni field when using the beacons strategy.
	X509VerifyHostname string
}

// IPAddr implements HTTPSDialerTactic.
func (t *circoTactic) IPAddr() string {
	return t.Address
}

// NewTLSHandshaker implements HTTPSDialerTactic.
func (t *circoTactic) NewTLSHandshaker(
	netx *netxlite.Netx, logger model.Logger) model.TLSHandshaker {
	return netxlite.NewTLSHandshakerStdlib(logger)
}

// InitialDelay implements HTTPSDialerTactic.
func (t *circoTactic) InitialDelay() time.Duration {
	return t.InitialWaitTime
}

// OnStarting implements HTTPSDialerTactic.
func (t *circoTactic) OnStarting() {
	// TODO(bassosimone): here we should collect metrics used to tweak
	// how we use beacons depending on the metrics
}

// OnSuccess implements HTTPSDialerTactic.
func (t *circoTactic) OnSuccess() {
	// TODO(bassosimone): here we should collect metrics used to tweak
	// how we use beacons depending on the metrics
}

// OnTCPConnectError implements HTTPSDialerTactic.
func (t *circoTactic) OnTCPConnectError(ctx context.Context, err error) {
	// TODO(bassosimone): here we should collect metrics used to tweak
	// how we use beacons depending on the metrics
}

// OnTLSHandshakeError implements HTTPSDialerTactic.
func (t *circoTactic) OnTLSHandshakeError(ctx context.Context, err error) {
	// TODO(bassosimone): here we should collect metrics used to tweak
	// how we use beacons depending on the metrics
}

// OnTLSVerifyError implements HTTPSDialerTactic.
func (*circoTactic) OnTLSVerifyError(ctx context.Context, err error) {
	// TODO(bassosimone): here we should collect metrics used to tweak
	// how we use beacons depending on the metrics
}

// SNI implements HTTPSDialerTactic.
func (t *circoTactic) SNI() string {
	return t.TLSServerName
}

// String implements HTTPSDialerTactic.
func (t *circoTactic) String() string {
	return fmt.Sprintf(
		"circoTactic{ipAddr:\"%s\" sni:\"%s\" verifyHostname:\"%s\" waitTime:%v}",
		t.Address, t.TLSServerName, t.X509VerifyHostname, t.InitialWaitTime,
	)
}

// VerifyHostname implements HTTPSDialerTactic.
func (t *circoTactic) VerifyHostname() string {
	return t.X509VerifyHostname
}
