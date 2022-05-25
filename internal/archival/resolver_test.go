package archival

import (
	"context"
	"errors"
	"io"
	"net"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/miekg/dns"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func TestSaverLookupHost(t *testing.T) {
	// newResolver helps to create a new resolver.
	newResolver := func(addrs []string, err error) model.Resolver {
		return &mocks.Resolver{
			MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
				return addrs, err
			},
			MockAddress: func() string {
				return "8.8.8.8:53"
			},
			MockNetwork: func() string {
				return "udp"
			},
		}
	}

	t.Run("on success", func(t *testing.T) {
		const domain = "dns.google"
		expectAddrs := []string{"8.8.8.8", "8.8.4.4"}
		saver := NewSaver()
		v := &SingleDNSLookupValidator{
			ExpectALPNs:           nil,
			ExpectAddrs:           expectAddrs,
			ExpectDomain:          domain,
			ExpectLookupType:      "getaddrinfo",
			ExpectFailure:         nil,
			ExpectResolverAddress: "8.8.8.8:53",
			ExpectResolverNetwork: "udp",
			Saver:                 saver,
		}
		reso := newResolver(expectAddrs, nil)
		ctx := context.Background()
		addrs, err := saver.LookupHost(ctx, reso, domain)
		if err != nil {
			t.Fatal(err)
		}
		if diff := cmp.Diff(expectAddrs, addrs); diff != "" {
			t.Fatal(diff)
		}
		if err := v.Validate(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("on failure", func(t *testing.T) {
		mockedError := netxlite.NewTopLevelGenericErrWrapper(io.EOF)
		const domain = "dns.google"
		saver := NewSaver()
		v := &SingleDNSLookupValidator{
			ExpectALPNs:           nil,
			ExpectAddrs:           nil,
			ExpectDomain:          domain,
			ExpectLookupType:      "getaddrinfo",
			ExpectFailure:         mockedError,
			ExpectResolverAddress: "8.8.8.8:53",
			ExpectResolverNetwork: "udp",
			Saver:                 saver,
		}
		reso := newResolver(nil, mockedError)
		ctx := context.Background()
		addrs, err := saver.LookupHost(ctx, reso, domain)
		if !errors.Is(err, mockedError) {
			t.Fatal("invalid err", err)
		}
		if len(addrs) != 0 {
			t.Fatal("invalid addrs")
		}
		if err := v.Validate(); err != nil {
			t.Fatal(err)
		}
	})
}

func TestSaverLookupHTTPS(t *testing.T) {
	// newResolver helps to create a new resolver.
	newResolver := func(alpns, ipv4, ipv6 []string, err error) model.Resolver {
		return &mocks.Resolver{
			MockLookupHTTPS: func(ctx context.Context, domain string) (*model.HTTPSSvc, error) {
				if alpns == nil && ipv4 == nil && ipv6 == nil {
					return nil, err
				}
				return &model.HTTPSSvc{
					ALPN: alpns,
					IPv4: ipv4,
					IPv6: ipv6,
				}, err
			},
			MockAddress: func() string {
				return "8.8.8.8:53"
			},
			MockNetwork: func() string {
				return "udp"
			},
		}
	}

	t.Run("on success", func(t *testing.T) {
		const domain = "dns.google"
		expectALPN := []string{"h3", "h2", "http/1.1"}
		expectA := []string{"8.8.8.8", "8.8.4.4"}
		expectAAAA := []string{"2001:4860:4860::8844"}
		expectAddrs := append(expectA, expectAAAA...)
		saver := NewSaver()
		v := &SingleDNSLookupValidator{
			ExpectALPNs:           expectALPN,
			ExpectAddrs:           expectAddrs,
			ExpectDomain:          domain,
			ExpectLookupType:      "https",
			ExpectFailure:         nil,
			ExpectResolverAddress: "8.8.8.8:53",
			ExpectResolverNetwork: "udp",
			Saver:                 saver,
		}
		reso := newResolver(expectALPN, expectA, expectAAAA, nil)
		ctx := context.Background()
		https, err := saver.LookupHTTPS(ctx, reso, domain)
		if err != nil {
			t.Fatal(err)
		}
		if diff := cmp.Diff(expectALPN, https.ALPN); diff != "" {
			t.Fatal(diff)
		}
		if diff := cmp.Diff(expectA, https.IPv4); diff != "" {
			t.Fatal(diff)
		}
		if diff := cmp.Diff(expectAAAA, https.IPv6); diff != "" {
			t.Fatal(diff)
		}
		if err := v.Validate(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("on failure", func(t *testing.T) {
		mockedError := netxlite.NewTopLevelGenericErrWrapper(io.EOF)
		const domain = "dns.google"
		saver := NewSaver()
		v := &SingleDNSLookupValidator{
			ExpectALPNs:           nil,
			ExpectAddrs:           nil,
			ExpectDomain:          domain,
			ExpectLookupType:      "https",
			ExpectFailure:         mockedError,
			ExpectResolverAddress: "8.8.8.8:53",
			ExpectResolverNetwork: "udp",
			Saver:                 saver,
		}
		reso := newResolver(nil, nil, nil, mockedError)
		ctx := context.Background()
		https, err := saver.LookupHTTPS(ctx, reso, domain)
		if !errors.Is(err, mockedError) {
			t.Fatal("unexpected err", err)
		}
		if https != nil {
			t.Fatal("expected nil https")
		}
		if err := v.Validate(); err != nil {
			t.Fatal(err)
		}
	})
}

type SingleDNSLookupValidator struct {
	ExpectALPNs           []string
	ExpectAddrs           []string
	ExpectDomain          string
	ExpectLookupType      string
	ExpectFailure         error
	ExpectResolverAddress string
	ExpectResolverNetwork string
	Saver                 *Saver
}

func (v *SingleDNSLookupValidator) Validate() error {
	trace := v.Saver.MoveOutTrace()
	var entries []*DNSLookupEvent
	switch v.ExpectLookupType {
	case "getaddrinfo":
		entries = trace.DNSLookupHost
	case "https":
		entries = trace.DNSLookupHTTPS
	default:
		return errors.New("invalid v.ExpectLookupType")
	}
	if len(entries) != 1 {
		return errors.New("expected a single entry")
	}
	entry := entries[0]
	if diff := cmp.Diff(v.ExpectALPNs, entry.ALPNs); diff != "" {
		return errors.New(diff)
	}
	if diff := cmp.Diff(v.ExpectAddrs, entry.Addresses); diff != "" {
		return errors.New(diff)
	}
	if v.ExpectDomain != entry.Domain {
		return errors.New("invalid .Domain value")
	}
	if !errors.Is(entry.Failure, v.ExpectFailure) {
		return errors.New("invalid .Failure value")
	}
	if !entry.Finished.After(entry.Started) {
		return errors.New(".Finished is not after .Started")
	}
	if entry.ResolverAddress != v.ExpectResolverAddress {
		return errors.New("invalid .ResolverAddress value")
	}
	if entry.ResolverNetwork != v.ExpectResolverNetwork {
		return errors.New("invalid .ResolverNetwork value")
	}
	return nil
}

func TestSaverDNSRoundTrip(t *testing.T) {
	// generateQueryAndResponse generates a fake query and reply.
	generateQueryAndResponse := func() (*mocks.DNSQuery, *mocks.DNSResponse) {
		queryID := dns.Id()
		query := &mocks.DNSQuery{
			MockDomain: func() string {
				return "x.org"
			},
			MockType: func() uint16 {
				return dns.TypeA
			},
			MockBytes: func() ([]byte, error) {
				return []byte{0xde, 0xad, 0xbe, 0xff}, nil
			},
			MockID: func() uint16 {
				return queryID
			},
		}
		response := &mocks.DNSResponse{
			MockQuery: func() model.DNSQuery {
				return query
			},
			MockBytes: func() []byte {
				return []byte{0xff, 0xbe, 0xad, 0xde}
			},
			MockRcode: func() int {
				return 0
			},
			MockDecodeHTTPS: func() (*model.HTTPSSvc, error) {
				return nil, netxlite.ErrOODNSNoAnswer
			},
			MockDecodeLookupHost: func() ([]string, error) {
				return nil, netxlite.ErrOODNSNoAnswer
			},
			MockDecodeNS: func() ([]*net.NS, error) {
				return nil, netxlite.ErrOODNSNoAnswer
			},
		}
		return query, response
	}

	// newDNSTransport creates a suitable DNSTransport.
	newDNSTransport := func(reply *mocks.DNSResponse, err error) model.DNSTransport {
		return &mocks.DNSTransport{
			MockRoundTrip: func(ctx context.Context, query model.DNSQuery) (model.DNSResponse, error) {
				if err != nil {
					return nil, err
				}
				return reply, nil
			},
			MockNetwork: func() string {
				return "udp"
			},
			MockAddress: func() string {
				return "8.8.8.8:53"
			},
		}
	}

	t.Run("on success", func(t *testing.T) {
		query, expectedResponse := generateQueryAndResponse()
		saver := NewSaver()
		v := &SingleDNSRoundTripValidator{
			ExpectAddress:  "8.8.8.8:53",
			ExpectFailure:  nil,
			ExpectNetwork:  "udp",
			ExpectQuery:    query,
			ExpectResponse: expectedResponse,
			Saver:          saver,
		}
		ctx := context.Background()
		txp := newDNSTransport(expectedResponse, nil)
		response, err := saver.DNSRoundTrip(ctx, txp, query)
		if err != nil {
			t.Fatal(err)
		}
		if response == nil {
			t.Fatal("expected non nil response")
		}
		if err := v.Validate(); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("on failure", func(t *testing.T) {
		mockedError := netxlite.NewTopLevelGenericErrWrapper(io.EOF)
		query, _ := generateQueryAndResponse()
		saver := NewSaver()
		v := &SingleDNSRoundTripValidator{
			ExpectAddress:  "8.8.8.8:53",
			ExpectFailure:  mockedError,
			ExpectNetwork:  "udp",
			ExpectQuery:    query,
			ExpectResponse: nil,
			Saver:          saver,
		}
		ctx := context.Background()
		txp := newDNSTransport(nil, mockedError)
		response, err := saver.DNSRoundTrip(ctx, txp, query)
		if !errors.Is(err, mockedError) {
			t.Fatal(err)
		}
		if response != nil {
			t.Fatal("unexpected reply")
		}
		if err := v.Validate(); err != nil {
			t.Fatal(err)
		}
	})
}

type SingleDNSRoundTripValidator struct {
	ExpectAddress  string
	ExpectFailure  error
	ExpectNetwork  string
	ExpectQuery    model.DNSQuery
	ExpectResponse model.DNSResponse
	Saver          *Saver
}

func (v *SingleDNSRoundTripValidator) Validate() error {
	trace := v.Saver.MoveOutTrace()
	if len(trace.DNSRoundTrip) != 1 {
		return errors.New("expected a single entry")
	}
	entry := trace.DNSRoundTrip[0]
	if v.ExpectAddress != entry.Address {
		return errors.New("invalid .Address")
	}
	if !errors.Is(entry.Failure, v.ExpectFailure) {
		return errors.New("invalid .Failure value")
	}
	if !entry.Finished.After(entry.Started) {
		return errors.New(".Finished is not after .Started")
	}
	if v.ExpectNetwork != entry.Network {
		return errors.New("invalid .Network value")
	}
	rawQuery, err := v.ExpectQuery.Bytes()
	if err != nil {
		return err
	}
	if diff := cmp.Diff(rawQuery, entry.Query); diff != "" {
		return errors.New(diff)
	}
	if v.ExpectResponse == nil {
		if entry.Reply != nil {
			return errors.New("reply is not nil")
		}
		return nil
	}
	if diff := cmp.Diff(v.ExpectResponse.Bytes(), entry.Reply); diff != "" {
		return errors.New(diff)
	}
	return nil
}
