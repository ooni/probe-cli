package mocks

import "context"

// DNSTransport allows mocking dnsx.DNSTransport.
type DNSTransport struct {
	MockRoundTrip func(ctx context.Context, query []byte) (reply []byte, err error)

	MockRequiresPadding func() bool

	MockNetwork func() string

	MockAddress func() string

	MockCloseIdleConnections func()
}

// RoundTrip calls MockRoundTrip.
func (txp *DNSTransport) RoundTrip(ctx context.Context, query []byte) (reply []byte, err error) {
	return txp.MockRoundTrip(ctx, query)
}

// RequiresPadding calls MockRequiresPadding.
func (txp *DNSTransport) RequiresPadding() bool {
	return txp.MockRequiresPadding()
}

// Network calls MockNetwork.
func (txp *DNSTransport) Network() string {
	return txp.MockNetwork()
}

// Address calls MockAddress.
func (txp *DNSTransport) Address() string {
	return txp.MockAddress()
}

// CloseIdleConnections calls MockCloseIdleConnections.
func (txp *DNSTransport) CloseIdleConnections() {
	txp.MockCloseIdleConnections()
}
