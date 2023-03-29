// Package geolocate implements IP lookup, resolver lookup, and geolocation.
package geolocate

import (
	"context"
	"fmt"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/version"
)

// Results contains geolocate results.
type Results struct {
	// ASN is the autonomous system number.
	ASN uint

	// CountryCode is the country code.
	CountryCode string

	// didResolverLookup indicates whether we did a resolver lookup.
	didResolverLookup bool

	// NetworkName is the network name.
	NetworkName string

	// IPAddr is the probe IPAddr.
	IPAddr string

	// ResolverASNumber is the resolver ASN.
	ResolverASNumber uint

	// ResolverIPAddr is the resolver IP.
	ResolverIPAddr string

	// ResolverASNetworkName is the resolver network name.
	ResolverASNetworkName string
}

var _ model.LocationProvider = &Results{}

// ProbeASN implements model.LocationProvider
func (r *Results) ProbeASN() uint {
	return r.ASN
}

// ProbeASNString returns the ASN as a string.
func (r *Results) ProbeASNString() string {
	return fmt.Sprintf("AS%d", r.ASN)
}

// ProbeCC implements model.LocationProvider
func (r *Results) ProbeCC() string {
	return r.CountryCode
}

// ProbeIP implements model.LocationProvider
func (r *Results) ProbeIP() string {
	return r.IPAddr
}

// ProbeNetworkName implements model.LocationProvider
func (r *Results) ProbeNetworkName() string {
	return r.NetworkName
}

// ResolverASN implements model.LocationProvider
func (r *Results) ResolverASN() uint {
	return r.ResolverASNumber
}

// ResolverASNString implements model.LocationProvider
func (r *Results) ResolverASNString() string {
	return fmt.Sprintf("AS%d", r.ResolverASNumber)
}

// ResolverIP implements model.LocationProvider
func (r *Results) ResolverIP() string {
	return r.ResolverIPAddr
}

// ResolverNetworkName implements model.LocationProvider
func (r *Results) ResolverNetworkName() string {
	return r.ResolverASNetworkName
}

type probeIPLookupper interface {
	LookupProbeIP(ctx context.Context) (addr string, err error)
}

type asnLookupper interface {
	LookupASN(ip string) (asn uint, network string, err error)
}

type countryLookupper interface {
	LookupCC(ip string) (cc string, err error)
}

type resolverIPLookupper interface {
	LookupResolverIP(ctx context.Context) (addr string, err error)
}

// Config contains configuration for a geolocate Task.
type Config struct {
	// Resolver is the resolver we should use when
	// making requests for discovering the IP. When
	// this field is not set, we use the stdlib.
	Resolver model.Resolver

	// Logger is the logger to use. If not set, then we will
	// use a logger that discards all messages.
	Logger model.Logger

	// UserAgent is the user agent to use. If not set, then
	// we will use a default user agent.
	UserAgent string
}

// NewTask creates a new instance of Task from config.
func NewTask(config Config) *Task {
	if config.Logger == nil {
		config.Logger = model.DiscardLogger
	}
	if config.UserAgent == "" {
		config.UserAgent = fmt.Sprintf("ooniprobe-engine/%s", version.Version)
	}
	if config.Resolver == nil {
		config.Resolver = netxlite.NewStdlibResolver(config.Logger)
	}
	return &Task{
		countryLookupper:     mmdbLookupper{},
		logger:               config.Logger,
		probeIPLookupper:     ipLookupClient(config),
		probeASNLookupper:    mmdbLookupper{},
		resolverASNLookupper: mmdbLookupper{},
		resolverIPLookupper: resolverLookupClient{
			Logger: config.Logger,
		},
	}
}

// Task performs a geolocation. You must create a new
// instance of Task using the NewTask factory.
type Task struct {
	countryLookupper     countryLookupper
	logger               model.Logger
	probeIPLookupper     probeIPLookupper
	probeASNLookupper    asnLookupper
	resolverASNLookupper asnLookupper
	resolverIPLookupper  resolverIPLookupper
}

// Run runs the task.
func (op Task) Run(ctx context.Context) (*Results, error) {
	var err error
	out := &Results{
		ASN:                   model.DefaultProbeASN,
		CountryCode:           model.DefaultProbeCC,
		NetworkName:           model.DefaultProbeNetworkName,
		IPAddr:                model.DefaultProbeIP,
		ResolverASNumber:      model.DefaultResolverASN,
		ResolverIPAddr:        model.DefaultResolverIP,
		ResolverASNetworkName: model.DefaultResolverNetworkName,
	}
	ip, err := op.probeIPLookupper.LookupProbeIP(ctx)
	if err != nil {
		return out, fmt.Errorf("lookupProbeIP failed: %w", err)
	}
	out.IPAddr = ip
	asn, networkName, err := op.probeASNLookupper.LookupASN(out.IPAddr)
	if err != nil {
		return out, fmt.Errorf("lookupASN failed: %w", err)
	}
	op.logger.Infof("geolocate: probe ASN: %d", asn)
	op.logger.Infof("geolocate: probe network name: %s", networkName)
	out.ASN = asn
	out.NetworkName = networkName
	cc, err := op.countryLookupper.LookupCC(out.IPAddr)
	if err != nil {
		return out, fmt.Errorf("lookupProbeCC failed: %w", err)
	}
	op.logger.Infof("geolocate: country code: %s", cc)
	out.CountryCode = cc
	out.didResolverLookup = true
	// Note: ignoring the result of lookupResolverIP and lookupASN
	// here is intentional. We don't want this (~minor) failure
	// to influence the result of the overall lookup. Another design
	// here could be that of retrying the operation N times?
	resolverIP, err := op.resolverIPLookupper.LookupResolverIP(ctx)
	if err != nil {
		return out, nil // intentional
	}
	out.ResolverIPAddr = resolverIP
	resolverASN, resolverNetworkName, err := op.resolverASNLookupper.LookupASN(
		out.ResolverIPAddr,
	)
	if err != nil {
		return out, nil // intentional
	}
	op.logger.Infof("geolocate: resolver ASN: %d", resolverASN)
	op.logger.Infof("geolocate: resolver network name: %s", resolverNetworkName)
	out.ResolverASNumber = resolverASN
	out.ResolverASNetworkName = resolverNetworkName
	return out, nil
}
