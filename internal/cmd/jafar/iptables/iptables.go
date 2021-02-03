// Package iptables contains code for managing firewall rules. This package
// really only works reliably on Linux. In all other systems the functionality
// in here is just a set of stubs returning errors.
package iptables

import (
	"github.com/ooni/probe-cli/v3/internal/engine/runtimex"
)

type shell interface {
	createChains() error
	dropIfDestinationEquals(ip string) error
	rstIfDestinationEqualsAndIsTCP(ip string) error
	dropIfContainsKeywordHex(keyword string) error
	dropIfContainsKeyword(keyword string) error
	rstIfContainsKeywordHexAndIsTCP(keyword string) error
	rstIfContainsKeywordAndIsTCP(keyword string) error
	hijackDNS(address string) error
	hijackHTTPS(address string) error
	hijackHTTP(address string) error
	waive() error
}

// CensoringPolicy implements a censoring policy.
type CensoringPolicy struct {
	DropIPs            []string // drop IP traffic to these IPs
	DropKeywordsHex    []string // drop IP packets with these hex keywords
	DropKeywords       []string // drop IP packets with these keywords
	HijackDNSAddress   string   // where to hijack DNS to
	HijackHTTPSAddress string   // where to hijack HTTPS to
	HijackHTTPAddress  string   // where to hijack HTTP to
	ResetIPs           []string // RST TCP/IP traffic to these IPs
	ResetKeywordsHex   []string // RST TCP/IP flows with these hex keywords
	ResetKeywords      []string // RST TCP/IP flows with these keywords
	sh                 shell
}

// NewCensoringPolicy returns a new censoring policy.
func NewCensoringPolicy() *CensoringPolicy {
	return &CensoringPolicy{
		sh: newShell(),
	}
}

// Apply applies the censorship policy
func (c *CensoringPolicy) Apply() (err error) {
	defer func() {
		if recover() != nil {
			// JUST KNOW WE'VE BEEN HERE
		}
	}()
	err = c.sh.createChains()
	runtimex.PanicOnError(err, "c.sh.createChains failed")
	// Implementation note: we want the RST rules to be first such
	// that we end up enforcing them before the drop rules.
	for _, keyword := range c.ResetKeywordsHex {
		err = c.sh.rstIfContainsKeywordHexAndIsTCP(keyword)
		runtimex.PanicOnError(err, "c.sh.rstIfContainsKeywordHexAndIsTCP failed")
	}
	for _, keyword := range c.ResetKeywords {
		err = c.sh.rstIfContainsKeywordAndIsTCP(keyword)
		runtimex.PanicOnError(err, "c.sh.rstIfContainsKeywordAndIsTCP failed")
	}
	for _, ip := range c.ResetIPs {
		err = c.sh.rstIfDestinationEqualsAndIsTCP(ip)
		runtimex.PanicOnError(err, "c.sh.rstIfDestinationEqualsAndIsTCP failed")
	}
	for _, keyword := range c.DropKeywordsHex {
		err = c.sh.dropIfContainsKeywordHex(keyword)
		runtimex.PanicOnError(err, "c.sh.dropIfContainsKeywordHex failed")
	}
	for _, keyword := range c.DropKeywords {
		err = c.sh.dropIfContainsKeyword(keyword)
		runtimex.PanicOnError(err, "c.sh.dropIfContainsKeyword failed")
	}
	for _, ip := range c.DropIPs {
		err = c.sh.dropIfDestinationEquals(ip)
		runtimex.PanicOnError(err, "c.sh.dropIfDestinationEquals failed")
	}
	if c.HijackDNSAddress != "" {
		err = c.sh.hijackDNS(c.HijackDNSAddress)
		runtimex.PanicOnError(err, "c.sh.hijackDNS failed")
	}
	if c.HijackHTTPSAddress != "" {
		err = c.sh.hijackHTTPS(c.HijackHTTPSAddress)
		runtimex.PanicOnError(err, "c.sh.hijackHTTPS failed")
	}
	if c.HijackHTTPAddress != "" {
		err = c.sh.hijackHTTP(c.HijackHTTPAddress)
		runtimex.PanicOnError(err, "c.sh.hijackHTTP failed")
	}
	return
}

// Waive removes any censorship policy
func (c *CensoringPolicy) Waive() error {
	return c.sh.waive()
}
