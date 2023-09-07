package testingx

import (
	"context"
	"os"
	"time"

	"github.com/miekg/dns"
	"github.com/ooni/netem"
)

// DNSRoundTripper performs DNS round trips.
type DNSRoundTripper interface {
	RoundTrip(ctx context.Context, req []byte) (resp []byte, err error)
}

// DNSRoundTripperFunc makes a func implement the [DNSRoundTripper] interface.
type DNSRoundTripperFunc func(ctx context.Context, req []byte) (resp []byte, err error)

var _ DNSRoundTripper = DNSRoundTripperFunc(nil)

// RoundTrip implements DNSRoundTripper.
func (fx DNSRoundTripperFunc) RoundTrip(ctx context.Context, req []byte) (resp []byte, err error) {
	return fx(ctx, req)
}

// NewDNSRoundTripperWithDNSConfig implements [DNSRroundTripper] using a [*netem.DNSConfig].
func NewDNSRoundTripperWithDNSConfig(config *netem.DNSConfig) DNSRoundTripper {
	return &dnsRoundTripperWithDNSConfig{config}
}

type dnsRoundTripperWithDNSConfig struct {
	config *netem.DNSConfig
}

// RoundTrip implements DNSRoundTripper.
func (rtx *dnsRoundTripperWithDNSConfig) RoundTrip(ctx context.Context, req []byte) (resp []byte, err error) {
	return netem.DNSServerRoundTrip(rtx.config, req)
}

// NewDNSRoundTripperEmptyRespnse is a [DNSRoundTripper] that always returns an empty response.
func NewDNSRoundTripperEmptyRespnse() DNSRoundTripper {
	return DNSRoundTripperFunc(func(ctx context.Context, rawReq []byte) (rawResp []byte, err error) {
		req := &dns.Msg{}
		if err := req.Unpack(rawReq); err != nil {
			return nil, err
		}
		resp := &dns.Msg{}
		resp.SetRcode(req, dns.RcodeSuccess)
		// without any additional RRs
		return resp.Pack()
	})
}

// NewDNSRoundTripperNXDOMAIN is a [DNSRoundTripper] that always returns NXDOMAIN.
func NewDNSRoundTripperNXDOMAIN() DNSRoundTripper {
	// An empty DNS config always causes a NXDOMAIN response
	return NewDNSRoundTripperWithDNSConfig(netem.NewDNSConfig())
}

// NewDNSRoundTripperRefused is a [DNSRoundTripper] that always returns refused.
func NewDNSRoundTripperRefused() DNSRoundTripper {
	return DNSRoundTripperFunc(func(ctx context.Context, rawReq []byte) (rawResp []byte, err error) {
		req := &dns.Msg{}
		if err := req.Unpack(rawReq); err != nil {
			return nil, err
		}
		resp := &dns.Msg{}
		resp.SetRcode(req, dns.RcodeRefused)
		return resp.Pack()
	})
}

// NewDNSRoundTripperSimulateTimeout is a [DNSRoundTripper] that sleeps for the given amount
// of time and then returns to the caller the given error.
func NewDNSRoundTripperSimulateTimeout(timeout time.Duration, err error) DNSRoundTripper {
	return DNSRoundTripperFunc(func(ctx context.Context, req []byte) (resp []byte, err error) {
		select {
		case <-time.After(timeout):
			return nil, os.ErrDeadlineExceeded
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	})
}
