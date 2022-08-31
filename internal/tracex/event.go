package tracex

//
// All the possible events
//

import (
	"errors"
	"net/http"
	"time"

	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// FailureStr is the string representation of an error. The empty
// string represents the absence of any error.
type FailureStr string

// NewFailureStr creates a FailureStr from an error. If the error is not
// already an ErrWrapper, it's converted to an ErrWrapper. If the ErrWrapper's
// Failure is not empty, we return that. Otherwise we return a string
// indicating that an ErrWrapper has an empty failure (should not happen).
func NewFailureStr(err error) FailureStr {
	if err == nil {
		return ""
	}
	// The following code guarantees that the error is always wrapped even
	// when we could not actually hit our code that does the wrapping. A case
	// in which this could happen is with context deadline for HTTP when you
	// have wrapped the underlying dialers but not the Transport.
	var errWrapper *netxlite.ErrWrapper
	if !errors.As(err, &errWrapper) {
		err := netxlite.NewTopLevelGenericErrWrapper(err)
		couldConvert := errors.As(err, &errWrapper)
		runtimex.Assert(couldConvert, "we should have an ErrWrapper here")
	}
	s := errWrapper.Failure
	if s == "" {
		s = "unknown_failure: errWrapper.Failure is empty"
	}
	return FailureStr(s)
}

// IsNil returns whether this FailureStr is nil. Technically speaking, the
// failure cannot be nil, but an empty string is equivalent to nil after
// we convert using ToFailure(). Also, this type is often called Err, Error,
// or Failure. So, the resulting code actually reads correct.
func (fs FailureStr) IsNil() bool {
	return fs == ""
}

// IsNotNil is the opposite of IsNil. Technically speaking, the
// failure cannot be nil, but an empty string is equivalent to nil after
// we convert using ToFailure(). Also, this type is often called Err, Error,
// or Failure. So, the resulting code actually reads correct.
func (fs FailureStr) IsNotNil() bool {
	return !fs.IsNil()
}

// ToFailure converts a FailureStr to a OONI failure (i.e., a string
// on error and nil in case of success).
func (fs FailureStr) ToFailure() (out *string) {
	if fs != "" {
		s := string(fs)
		out = &s
	}
	return
}

// Event is one of the events within a trace.
type Event interface {
	// Value returns the event value
	Value() *EventValue

	// Name returns the event name
	Name() string
}

// EventTLSHandshakeStart is the beginning of the TLS handshake.
type EventTLSHandshakeStart struct {
	V *EventValue
}

func (ev *EventTLSHandshakeStart) Value() *EventValue {
	return ev.V
}

func (ev *EventTLSHandshakeStart) Name() string {
	return "tls_handshake_start"
}

// EventTLSHandshakeDone is the end of the TLS handshake.
type EventTLSHandshakeDone struct {
	V *EventValue
}

func (ev *EventTLSHandshakeDone) Value() *EventValue {
	return ev.V
}

func (ev *EventTLSHandshakeDone) Name() string {
	return "tls_handshake_done"
}

// EventResolveStart is the beginning of a DNS lookup operation.
type EventResolveStart struct {
	V *EventValue
}

func (ev *EventResolveStart) Value() *EventValue {
	return ev.V
}

func (ev *EventResolveStart) Name() string {
	return "resolve_start"
}

// EventResolveDone is the end of a DNS lookup operation.
type EventResolveDone struct {
	V *EventValue
}

func (ev *EventResolveDone) Value() *EventValue {
	return ev.V
}

func (ev *EventResolveDone) Name() string {
	return "resolve_done"
}

// EventDNSRoundTripStart is the start of a DNS round trip.
type EventDNSRoundTripStart struct {
	V *EventValue
}

func (ev *EventDNSRoundTripStart) Value() *EventValue {
	return ev.V
}

func (ev *EventDNSRoundTripStart) Name() string {
	return "dns_round_trip_start"
}

// EventDNSRoundTripDone is the end of a DNS round trip.
type EventDNSRoundTripDone struct {
	V *EventValue
}

func (ev *EventDNSRoundTripDone) Value() *EventValue {
	return ev.V
}

func (ev *EventDNSRoundTripDone) Name() string {
	return "dns_round_trip_done"
}

// EventQUICHandshakeStart is the start of a QUIC handshake.
type EventQUICHandshakeStart struct {
	V *EventValue
}

func (ev *EventQUICHandshakeStart) Value() *EventValue {
	return ev.V
}

func (ev *EventQUICHandshakeStart) Name() string {
	return "quic_handshake_start"
}

// EventQUICHandshakeDone is the end of a QUIC handshake.
type EventQUICHandshakeDone struct {
	V *EventValue
}

func (ev *EventQUICHandshakeDone) Value() *EventValue {
	return ev.V
}

func (ev *EventQUICHandshakeDone) Name() string {
	return "quic_handshake_done"
}

// EventWriteToOperation summarizes the WriteTo operation.
type EventWriteToOperation struct {
	V *EventValue
}

func (ev *EventWriteToOperation) Value() *EventValue {
	return ev.V
}

func (ev *EventWriteToOperation) Name() string {
	return netxlite.WriteToOperation
}

// EventReadFromOperation summarizes the ReadFrom operation.
type EventReadFromOperation struct {
	V *EventValue
}

func (ev *EventReadFromOperation) Value() *EventValue {
	return ev.V
}

func (ev *EventReadFromOperation) Name() string {
	return netxlite.ReadFromOperation
}

// EventHTTPTransactionStart is the beginning of an HTTP transaction.
type EventHTTPTransactionStart struct {
	V *EventValue
}

func (ev *EventHTTPTransactionStart) Value() *EventValue {
	return ev.V
}

func (ev *EventHTTPTransactionStart) Name() string {
	return "http_transaction_start"
}

// EventHTTPTransactionDone is the end of an HTTP transaction.
type EventHTTPTransactionDone struct {
	V *EventValue
}

func (ev *EventHTTPTransactionDone) Value() *EventValue {
	return ev.V
}

func (ev *EventHTTPTransactionDone) Name() string {
	return "http_transaction_done"
}

// EventConnectOperation contains information about the connect operation.
type EventConnectOperation struct {
	V *EventValue
}

func (ev *EventConnectOperation) Value() *EventValue {
	return ev.V
}

func (ev *EventConnectOperation) Name() string {
	return netxlite.ConnectOperation
}

// EventReadOperation contains information about a read operation.
type EventReadOperation struct {
	V *EventValue
}

func (ev *EventReadOperation) Value() *EventValue {
	return ev.V
}

func (ev *EventReadOperation) Name() string {
	return netxlite.ReadOperation
}

// EventWriteOperation contains information about a write operation.
type EventWriteOperation struct {
	V *EventValue
}

func (ev *EventWriteOperation) Value() *EventValue {
	return ev.V
}

func (ev *EventWriteOperation) Name() string {
	return netxlite.WriteOperation
}

// Event is one of the events within a trace
type EventValue struct {
	Addresses                   []string      `json:",omitempty"`
	Address                     string        `json:",omitempty"`
	DNSQuery                    []byte        `json:",omitempty"`
	DNSResponse                 []byte        `json:",omitempty"`
	Data                        []byte        `json:",omitempty"`
	Duration                    time.Duration `json:",omitempty"`
	Err                         FailureStr    `json:",omitempty"`
	HTTPMethod                  string        `json:",omitempty"`
	HTTPRequestHeaders          http.Header   `json:",omitempty"`
	HTTPResponseHeaders         http.Header   `json:",omitempty"`
	HTTPResponseBody            []byte        `json:",omitempty"`
	HTTPResponseBodyIsTruncated bool          `json:",omitempty"`
	HTTPStatusCode              int           `json:",omitempty"`
	HTTPURL                     string        `json:",omitempty"`
	Hostname                    string        `json:",omitempty"`
	NoTLSVerify                 bool          `json:",omitempty"`
	NumBytes                    int           `json:",omitempty"`
	Proto                       string        `json:",omitempty"`
	TLSServerName               string        `json:",omitempty"`
	TLSCipherSuite              string        `json:",omitempty"`
	TLSNegotiatedProto          string        `json:",omitempty"`
	TLSNextProtos               []string      `json:",omitempty"`
	TLSPeerCerts                [][]byte      `json:",omitempty"`
	TLSVersion                  string        `json:",omitempty"`
	Time                        time.Time     `json:",omitempty"`
	Transport                   string        `json:",omitempty"`
}
