package urlgetter

import (
	"crypto/tls"
	"errors"
	"net"
	"net/url"
	"regexp"
	"strings"

	"github.com/ooni/probe-cli/v3/internal/engine/netx"
	"github.com/ooni/probe-cli/v3/internal/tracex"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// The Configurer job is to construct a Configuration that can
// later be used by the measurer to perform measurements.
type Configurer struct {
	Config   Config
	Logger   model.Logger
	ProxyURL *url.URL
	Saver    *tracex.Saver
}

// The Configuration is the configuration for running a measurement.
type Configuration struct {
	HTTPConfig netx.Config
	DNSClient  model.Resolver
}

// CloseIdleConnections will close idle connections, if needed.
func (c Configuration) CloseIdleConnections() {
	c.DNSClient.CloseIdleConnections()
}

// NewConfiguration builds a new measurement configuration.
func (c Configurer) NewConfiguration() (Configuration, error) {
	// set up defaults
	configuration := Configuration{
		HTTPConfig: netx.Config{
			BogonIsError:        c.Config.RejectDNSBogons,
			CacheResolutions:    true,
			CertPool:            c.Config.CertPool,
			ContextByteCounting: true,
			DialSaver:           c.Saver,
			HTTP3Enabled:        c.Config.HTTP3Enabled,
			HTTPSaver:           c.Saver,
			Logger:              c.Logger,
			ReadWriteSaver:      c.Saver,
			ResolveSaver:        c.Saver,
			TLSSaver:            c.Saver,
		},
	}
	// fill DNS cache
	if c.Config.DNSCache != "" {
		entry := strings.Split(c.Config.DNSCache, " ")
		if len(entry) < 2 {
			return configuration, errors.New("invalid DNSCache string")
		}
		domainregex := regexp.MustCompile(`^([a-z0-9]+(-[a-z0-9]+)*\.)+[a-z]{2,}$`)
		if !domainregex.MatchString(entry[0]) {
			return configuration, errors.New("invalid domain in DNSCache")
		}
		var addresses []string
		for i := 1; i < len(entry); i++ {
			if net.ParseIP(entry[i]) == nil {
				return configuration, errors.New("invalid IP in DNSCache")
			}
			addresses = append(addresses, entry[i])
		}
		configuration.HTTPConfig.DNSCache = map[string][]string{
			entry[0]: addresses,
		}
	}
	dnsclient, err := netx.NewDNSClientWithOverrides(
		configuration.HTTPConfig, c.Config.ResolverURL,
		c.Config.DNSHTTPHost, c.Config.DNSTLSServerName,
		c.Config.DNSTLSVersion,
	)
	if err != nil {
		return configuration, err
	}
	configuration.DNSClient = dnsclient
	configuration.HTTPConfig.BaseResolver = dnsclient
	// configure TLS
	configuration.HTTPConfig.TLSConfig = &tls.Config{
		NextProtos: []string{"h2", "http/1.1"},
	}
	if c.Config.TLSServerName != "" {
		configuration.HTTPConfig.TLSConfig.ServerName = c.Config.TLSServerName
	}
	err = netxlite.ConfigureTLSVersion(
		configuration.HTTPConfig.TLSConfig, c.Config.TLSVersion,
	)
	if err != nil {
		return configuration, err
	}
	configuration.HTTPConfig.NoTLSVerify = c.Config.NoTLSVerify
	// configure proxy
	configuration.HTTPConfig.ProxyURL = c.ProxyURL
	return configuration, nil
}
