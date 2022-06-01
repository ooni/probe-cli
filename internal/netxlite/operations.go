package netxlite

//
// Names of operations
//

// Operations that we measure. They are the possible values of
// the ErrWrapper.Operation field.
const (
	// ResolveOperation is the operation where we resolve a domain name.
	ResolveOperation = "resolve"

	// ConnectOperation is the operation where we do a TCP connect.
	ConnectOperation = "connect"

	// DNSRoundTripOperation is the DNS round trip.
	DNSRoundTripOperation = "dns_round_trip"

	// TLSHandshakeOperation is the TLS handshake.
	TLSHandshakeOperation = "tls_handshake"

	// QUICHandshakeOperation is the handshake to setup a QUIC connection.
	QUICHandshakeOperation = "quic_handshake"

	// QUICListenOperation is when we open a listening UDP conn for QUIC.
	QUICListenOperation = "quic_listen"

	// HTTPRoundTripOperation is the HTTP round trip.
	HTTPRoundTripOperation = "http_round_trip"

	// CloseOperation is when we close a socket.
	CloseOperation = "close"

	// ReadOperation is when we read from a socket.
	ReadOperation = "read"

	// WriteOperation is when we write to a socket.
	WriteOperation = "write"

	// ReadFromOperation is when we read from an UDP socket.
	ReadFromOperation = "read_from"

	// WriteToOperation is when we write to an UDP socket.
	WriteToOperation = "write_to"

	// UnknownOperation is when we cannot determine the operation.
	UnknownOperation = "unknown"

	// TopLevelOperation is used when the failure happens at top level. This
	// happens for example with urlgetter with a cancelled context.
	TopLevelOperation = "top_level"
)
