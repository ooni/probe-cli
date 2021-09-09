package mocks

import "context"

// RoundTripper allows mocking dnsx.RoundTripper.
type RoundTripper struct {
	MockRoundTrip func(ctx context.Context, query []byte) (reply []byte, err error)

	MockRequiresPadding func() bool

	MockNetwork func() string

	MockAddress func() string

	MockCloseIdleConnections func()
}

// RoundTrip calls MockRoundTrip.
func (txp *RoundTripper) RoundTrip(ctx context.Context, query []byte) (reply []byte, err error) {
	return txp.MockRoundTrip(ctx, query)
}

// RequiresPadding calls MockRequiresPadding.
func (txp *RoundTripper) RequiresPadding() bool {
	return txp.MockRequiresPadding()
}

// Network calls MockNetwork.
func (txp *RoundTripper) Network() string {
	return txp.MockNetwork()
}

// Address calls MockAddress.
func (txp *RoundTripper) Address() string {
	return txp.MockAddress()
}

// CloseIdleConnections calls MockCloseIdleConnections.
func (txp *RoundTripper) CloseIdleConnections() {
	txp.MockCloseIdleConnections()
}
