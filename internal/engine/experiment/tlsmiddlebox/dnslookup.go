package tlsmiddlebox

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/tracex"
)

// DNSLookUp resolves the input domain and outputs a model.ArchivalDNSLookUpResult
// SUGGESTION (Simone): Using Firefox's TRR2 resolver
func (m *Measurer) DNSLookup(ctx context.Context,
	domain string, r model.Resolver) (*model.ArchivalDNSLookupResult, []string, error) {
	url := m.config.resolverURL()
	logger := model.DiscardLogger
	resolver := NewResolver(r, logger, url)
	addrs, err := resolver.LookupHost(ctx, domain)
	if err != nil {
		systemResolver := NewResolver(r, logger, "")
		addrs, err = systemResolver.LookupHost(ctx, domain)
		out := WriteDNSToArchival(systemResolver, domain, addrs, err)
		if err != nil {
			out.Failure = tracex.NewFailure(err)
		}
		return out, addrs, err
	}
	out := WriteDNSToArchival(resolver, domain, addrs, err)
	return out, addrs, err
}

// WriteDNSToArchival populates the model.ArchivalDNSLookUpResult
func WriteDNSToArchival(resolver model.Resolver, domain string, addrs []string, err error) (out *model.ArchivalDNSLookupResult) {
	out = &model.ArchivalDNSLookupResult{
		Answers:         []model.ArchivalDNSAnswer{},
		Failure:         tracex.NewFailure(err),
		Engine:          resolver.Network(),
		Hostname:        domain,
		ResolverAddress: resolver.Address(),
	}
	answers := answersFromAddrs(addrs)
	out.Answers = append(out.Answers, answers...)
	return
}

// answerFromAddrs populates the model.ArchivalDNSAnswer using an array of addresses
func answersFromAddrs(addrs []string) (out []model.ArchivalDNSAnswer) {
	out = []model.ArchivalDNSAnswer{}
	for _, addr := range addrs {
		ipv6, err := netxlite.IsIPv6(addr)
		if err != nil {
			continue
		}
		switch ipv6 {
		case false:
			out = append(out, model.ArchivalDNSAnswer{
				AnswerType: "A",
				IPv4:       addr,
			})
		case true:
			out = append(out, model.ArchivalDNSAnswer{
				AnswerType: "AAAA",
				IPv6:       addr,
			})
		}
	}
	return
}

// NewResolver returns the passed resolver if not nil
// If the passed resolver is nil, a DoH or system resolver depending on the passed url
func NewResolver(r model.Resolver, logger model.DebugLogger, url string) model.Resolver {
	if r != nil {
		return r
	}
	if url != "" {
		return netxlite.NewParallelDNSOverHTTPSResolver(logger, url)
	}
	return netxlite.NewStdlibResolver(logger)
}
