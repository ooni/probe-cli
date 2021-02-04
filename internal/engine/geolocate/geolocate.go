// Package geolocate implements IP lookup, resolver lookup, and geolocation.
package geolocate

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/ooni/probe-cli/v3/internal/engine/model"
	"github.com/ooni/probe-cli/v3/internal/engine/runtimex"
	"github.com/ooni/probe-cli/v3/internal/version"
)

const (
	// DefaultProbeASN is the default probe ASN as number.
	DefaultProbeASN uint = 0

	// DefaultProbeCC is the default probe CC.
	DefaultProbeCC = "ZZ"

	// DefaultProbeIP is the default probe IP.
	DefaultProbeIP = model.DefaultProbeIP

	// DefaultProbeNetworkName is the default probe network name.
	DefaultProbeNetworkName = ""

	// DefaultResolverASN is the default resolver ASN.
	DefaultResolverASN uint = 0

	// DefaultResolverIP is the default resolver IP.
	DefaultResolverIP = "127.0.0.2"

	// DefaultResolverNetworkName is the default resolver network name.
	DefaultResolverNetworkName = ""
)

var (
	// DefaultProbeASNString is the default probe ASN as a string.
	DefaultProbeASNString = fmt.Sprintf("AS%d", DefaultProbeASN)

	// DefaultResolverASNString is the default resolver ASN as a string.
	DefaultResolverASNString = fmt.Sprintf("AS%d", DefaultResolverASN)
)

var (
	// ErrMissingResourcesManager indicates that no resources
	// manager has been configured inside of Config.
	ErrMissingResourcesManager = errors.New("geolocate: ResourcesManager is nil")
)

// Logger is the definition of Logger used by this package.
type Logger interface {
	Debugf(format string, v ...interface{})
}

// Results contains geolocate results
type Results struct {
	// ASN is the autonomous system number
	ASN uint

	// CountryCode is the country code
	CountryCode string

	// DidResolverLookup indicates whether we did a resolver lookup.
	DidResolverLookup bool

	// NetworkName is the network name
	NetworkName string

	// IP is the probe IP
	ProbeIP string

	// ResolverASN is the resolver ASN
	ResolverASN uint

	// ResolverIP is the resolver IP
	ResolverIP string

	// ResolverNetworkName is the resolver network name
	ResolverNetworkName string
}

// ASNString returns the ASN as a string
func (r *Results) ASNString() string {
	return fmt.Sprintf("AS%d", r.ASN)
}

type probeIPLookupper interface {
	LookupProbeIP(ctx context.Context) (addr string, err error)
}

type asnLookupper interface {
	LookupASN(path string, ip string) (asn uint, network string, err error)
}

type countryLookupper interface {
	LookupCC(path string, ip string) (cc string, err error)
}

type resolverIPLookupper interface {
	LookupResolverIP(ctx context.Context) (addr string, err error)
}

// ResourcesManager manages the required resources.
type ResourcesManager interface {
	// ASNDatabasePath returns the path of the ASN database.
	ASNDatabasePath() string

	// CountryDatabasePath returns the path of the country database.
	CountryDatabasePath() string

	// MaybeUpdateResources ensures that the required resources
	// have been downloaded and are current.
	MaybeUpdateResources(ctx context.Context) error
}

// Config contains configuration for a geolocate Task.
type Config struct {
	// EnableResolverLookup indicates whether we want to
	// perform the optional resolver lookup.
	EnableResolverLookup bool

	// HTTPClient is the HTTP client to use. If not set, then
	// we will use the http.DefaultClient.
	HTTPClient *http.Client

	// Logger is the logger to use. If not set, then we will
	// use a logger that discards all messages.
	Logger Logger

	// ResourcesManager is the mandatory resources manager. If not
	// set, we will not be able to perform any lookup.
	ResourcesManager ResourcesManager

	// UserAgent is the user agent to use. If not set, then
	// we will use a default user agent.
	UserAgent string
}

// Must ensures that NewTask is successful.
func Must(task *Task, err error) *Task {
	runtimex.PanicOnError(err, "NewTask failed")
	return task
}

// NewTask creates a new instance of Task from config.
func NewTask(config Config) (*Task, error) {
	if config.HTTPClient == nil {
		config.HTTPClient = http.DefaultClient
	}
	if config.Logger == nil {
		config.Logger = model.DiscardLogger
	}
	if config.ResourcesManager == nil {
		return nil, ErrMissingResourcesManager
	}
	if config.UserAgent == "" {
		config.UserAgent = fmt.Sprintf("ooniprobe-engine/%s", version.Version)
	}
	return &Task{
		countryLookupper:     mmdbLookupper{},
		enableResolverLookup: config.EnableResolverLookup,
		probeIPLookupper: ipLookupClient{
			HTTPClient: config.HTTPClient,
			Logger:     config.Logger,
			UserAgent:  config.UserAgent,
		},
		probeASNLookupper:    mmdbLookupper{},
		resolverASNLookupper: mmdbLookupper{},
		resolverIPLookupper:  resolverLookupClient{},
		resourcesManager:     config.ResourcesManager,
	}, nil
}

// Task performs a geolocation. You must create a new
// instance of Task using the NewTask factory.
type Task struct {
	countryLookupper     countryLookupper
	enableResolverLookup bool
	probeIPLookupper     probeIPLookupper
	probeASNLookupper    asnLookupper
	resolverASNLookupper asnLookupper
	resolverIPLookupper  resolverIPLookupper
	resourcesManager     ResourcesManager
}

// Run runs the task.
func (op Task) Run(ctx context.Context) (*Results, error) {
	var err error
	out := &Results{
		ASN:                 DefaultProbeASN,
		CountryCode:         DefaultProbeCC,
		NetworkName:         DefaultProbeNetworkName,
		ProbeIP:             DefaultProbeIP,
		ResolverASN:         DefaultResolverASN,
		ResolverIP:          DefaultResolverIP,
		ResolverNetworkName: DefaultResolverNetworkName,
	}
	if err := op.resourcesManager.MaybeUpdateResources(ctx); err != nil {
		return out, fmt.Errorf("MaybeUpdateResource failed: %w", err)
	}
	ip, err := op.probeIPLookupper.LookupProbeIP(ctx)
	if err != nil {
		return out, fmt.Errorf("lookupProbeIP failed: %w", err)
	}
	out.ProbeIP = ip
	asn, networkName, err := op.probeASNLookupper.LookupASN(
		op.resourcesManager.ASNDatabasePath(), out.ProbeIP)
	if err != nil {
		return out, fmt.Errorf("lookupASN failed: %w", err)
	}
	out.ASN = asn
	out.NetworkName = networkName
	cc, err := op.countryLookupper.LookupCC(
		op.resourcesManager.CountryDatabasePath(), out.ProbeIP)
	if err != nil {
		return out, fmt.Errorf("lookupProbeCC failed: %w", err)
	}
	out.CountryCode = cc
	if op.enableResolverLookup {
		out.DidResolverLookup = true
		// Note: ignoring the result of lookupResolverIP and lookupASN
		// here is intentional. We don't want this (~minor) failure
		// to influence the result of the overall lookup. Another design
		// here could be that of retrying the operation N times?
		resolverIP, err := op.resolverIPLookupper.LookupResolverIP(ctx)
		if err != nil {
			return out, nil
		}
		out.ResolverIP = resolverIP
		resolverASN, resolverNetworkName, err := op.resolverASNLookupper.LookupASN(
			op.resourcesManager.ASNDatabasePath(), out.ResolverIP,
		)
		if err != nil {
			return out, nil
		}
		out.ResolverASN = resolverASN
		out.ResolverNetworkName = resolverNetworkName
	}
	return out, nil
}
