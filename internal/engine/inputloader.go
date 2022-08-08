package engine

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/url"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/fsx"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/registry"
	"github.com/ooni/probe-cli/v3/internal/stuninput"
)

// These errors are returned by the InputLoader.
var (
	ErrNoURLsReturned    = errors.New("no URLs returned")
	ErrDetectedEmptyFile = errors.New("file did not contain any input")
	ErrInputRequired     = errors.New("no input provided")
	ErrNoInputExpected   = errors.New("we did not expect any input")
	ErrNoStaticInput     = errors.New("no static input for this experiment")
)

// InputLoaderSession is the session according to an InputLoader. We
// introduce this abstraction because it helps us with testing.
type InputLoaderSession interface {
	CheckIn(ctx context.Context,
		config *model.OOAPICheckInConfig) (*model.OOAPICheckInInfo, error)
}

// InputLoaderLogger is the logger according to an InputLoader.
type InputLoaderLogger interface {
	// Warnf formats and emits a warning message.
	Warnf(format string, v ...interface{})
}

// InputLoader loads input according to the specified policy
// either from command line and input files or from OONI services. The
// behaviour depends on the input policy as described below.
//
// You MUST NOT change any public field of this structure when
// in use, because that MAY lead to data races.
//
// InputNone
//
// We fail if there is any StaticInput or any SourceFiles. If
// there's no input, we return a single, empty entry that causes
// experiments that don't require input to run once.
//
// InputOptional
//
// We gather input from StaticInput and SourceFiles. If there is
// input, we return it. Otherwise we return a single, empty entry
// that causes experiments that don't require input to run once.
//
// InputOrQueryBackend
//
// We gather input from StaticInput and SourceFiles. If there is
// input, we return it. Otherwise, we use OONI's probe services
// to gather input using the best API for the task.
//
// InputOrStaticDefault
//
// We gather input from StaticInput and SourceFiles. If there is
// input, we return it. Otherwise, we return an internal static
// list of inputs to be used with this experiment.
//
// InputStrictlyRequired
//
// We gather input from StaticInput and SourceFiles. If there is
// input, we return it. Otherwise, we return an error.
type InputLoader struct {
	// CheckInConfig contains options for the CheckIn API. If
	// not set, then we'll create a default config. If set but
	// there are fields inside it that are not set, then we
	// will set them to a default value.
	CheckInConfig *model.OOAPICheckInConfig

	// ExperimentName is the name of the experiment. This field
	// is only used together with the InputOrStaticDefault policy.
	ExperimentName string

	// InputPolicy specifies the input policy for the
	// current experiment. We will not load any input if
	// the policy says we should not. You MUST fill in
	// this field.
	InputPolicy model.InputPolicy

	// Logger is the optional logger that the InputLoader
	// should be using. If not set, we will use the default
	// logger of github.com/apex/log.
	Logger InputLoaderLogger

	// Session is the current measurement session. You
	// MUST fill in this field.
	Session InputLoaderSession

	// StaticInputs contains optional input to be added
	// to the resulting input list if possible.
	StaticInputs []string

	// SourceFiles contains optional files to read input
	// from. Each file should contain a single input string
	// per line. We will fail if any file is unreadable
	// as well as if any file is empty.
	SourceFiles []string
}

// Load attempts to load input using the specified input loader. We will
// return a list of URLs because this is the only input we support.
func (il *InputLoader) Load(ctx context.Context) ([]model.OOAPIURLInfo, error) {
	switch il.InputPolicy {
	case model.InputOptional:
		return il.loadOptional()
	case model.InputOrQueryBackend:
		return il.loadOrQueryBackend(ctx)
	case model.InputStrictlyRequired:
		return il.loadStrictlyRequired(ctx)
	case model.InputOrStaticDefault:
		return il.loadOrStaticDefault(ctx)
	default:
		return il.loadNone()
	}
}

// loadNone implements the InputNone policy.
func (il *InputLoader) loadNone() ([]model.OOAPIURLInfo, error) {
	if len(il.StaticInputs) > 0 || len(il.SourceFiles) > 0 {
		return nil, ErrNoInputExpected
	}
	// Note that we need to return a single empty entry.
	return []model.OOAPIURLInfo{{}}, nil
}

// loadOptional implements the InputOptional policy.
func (il *InputLoader) loadOptional() ([]model.OOAPIURLInfo, error) {
	inputs, err := il.loadLocal()
	if err == nil && len(inputs) <= 0 {
		// Note that we need to return a single empty entry.
		inputs = []model.OOAPIURLInfo{{}}
	}
	return inputs, err
}

// loadStrictlyRequired implements the InputStrictlyRequired policy.
func (il *InputLoader) loadStrictlyRequired(ctx context.Context) ([]model.OOAPIURLInfo, error) {
	inputs, err := il.loadLocal()
	if err != nil || len(inputs) > 0 {
		return inputs, err
	}
	return nil, ErrInputRequired
}

// loadOrQueryBackend implements the InputOrQueryBackend policy.
func (il *InputLoader) loadOrQueryBackend(ctx context.Context) ([]model.OOAPIURLInfo, error) {
	inputs, err := il.loadLocal()
	if err != nil || len(inputs) > 0 {
		return inputs, err
	}
	return il.loadRemote(ctx)
}

// TODO(https://github.com/ooni/probe/issues/1390): we need to
// implement serving DNSCheck targets from the API
var dnsCheckDefaultInput = []string{
	"https://dns.google/dns-query",
	"https://8.8.8.8/dns-query",
	"dot://8.8.8.8:853/",
	"dot://8.8.4.4:853/",
	"https://8.8.4.4/dns-query",
	"https://cloudflare-dns.com/dns-query",
	"https://1.1.1.1/dns-query",
	"https://1.0.0.1/dns-query",
	"dot://1.1.1.1:853/",
	"dot://1.0.0.1:853/",
	"https://dns.quad9.net/dns-query",
	"https://9.9.9.9/dns-query",
	"dot://9.9.9.9:853/",
	"dot://dns.quad9.net/",
	"https://doh-de.blahdns.com/dns-query",
	"dot://doh-de.blahdns.com/dns-query",
	"https://dohtrial.att.net/dns-query",
	"dot://dohtrial.att.net/dns-query",
	"https://family.cloudflare-dns.com/dns-query",
	"dot://family.cloudflare-dns.com/dns-query",
	"https://doh.chi.ahadns.net/dns-query",
	"https://unicast.uncensoreddns.org/dns-query",
	"dot://unicast.uncensoreddns.org/dns-query",
	"https://doh.libredns.gr/dns-query",
	"dot://doh.libredns.gr/dns-query",
	"https://doh.eu.dnswarden.com/adblock",
	"https://doh-jp.blahdns.com/dns-query",
	"dot://doh-jp.blahdns.com/dns-query",
	"https://dnsnl.alekberg.net/dns-query",
	"https://doh.in.ahadns.net/dns-query",
	"https://doh.la.ahadns.net/dns-query",
	"https://doh-de.blahdns.com/dns-query",
	"dot://doh-de.blahdns.com/dns-query",
	"https://dnses.alekberg.net/dns-query",
	"https://doh.asia.dnswarden.com/adblock",
	"https://doh.ffmuc.net/dns-query",
	"dot://doh.ffmuc.net/dns-query",
	"https://dnsse.alekberg.net/dns-query",
	"https://dns.brahma.world/dns-query",
	"dot://dns.brahma.world/dns-query",
	"https://dns11.quad9.net/dns-query",
	"dot://dns11.quad9.net/dns-query",
	"https://doh.dnscrypt.uk/dns-query",
	"https://doh.appliedprivacy.ne/dns-query",
	"dot://doh.appliedprivacy.ne/dns-query",
	"https://dnsnl.alekberg.net/dns-query",
	"https://query.hdns.io/dns-query",
	"https://jp.tiar.app/dns-query",
	"dot://jp.tiar.app/dns-query",
	"https://dns.dnshome.de/dns-query",
	"dot://dns.dnshome.de/dns-query",
	"https://dns9.quad9.net/dns-query",
	"dot://dns9.quad9.net/dns-query",
	"https://dns12.quad9.net/dns-query",
	"dot://dns12.quad9.net/dns-query",
	"https://dns1.ryan-palmer.com/dns-query",
	"https://doh.pl.ahadns.net/dns-query",
	"https://jp.tiarap.org/dns-query",
	"https://odvr.nic.cz/dns-query",
	"dot://odvr.nic.cz/dns-query",
	"https://doh.us.dnswarden.com/adblock",
	"https://draco.plan9-ns2.com/dns-query",
	"dot://draco.plan9-ns2.com/dns-query",
	"https://freedns.controld.com/dns-query",
	"dot://freedns.controld.com/dns-query",
	"https://1dot1dot1dot1.cloudflare-dns.com/dns-query",
	"dot://1dot1dot1dot1.cloudflare-dns.com/dns-query",
	"https://dns.adguard.com/dns-query",
	"dot://dns.adguard.com/dns-query",
	"https://adfree.usableprivacy.net/dns-query",
	"https://dns-family.adguard.com/dns-query",
	"dot://dns-family.adguard.com/dns-query",
	"https://dns.circl.lu/dns-query",
	"https://doh-fi.blahdns.com/dns-query",
	"dot://doh-fi.blahdns.com/dns-query",
	"dot://doh.cleanbrowsing.org/",
	"https://dns.cloudflare.com/dns-query",
	"https://doh.tiar.app/dns-query",
	"dot://doh.tiar.app/dns-query",
	"https://doh-2.seby.io/dns-query",
	"https://public.dns.iij.jp/dns-query",
	"dot://public.dns.iij.jp/dns-query",
	"https://adblock.doh.mullvad.net/dns-query",
	"dot://adblock.doh.mullvad.net/dns-query",
	"https://dns.alidns.com/dns-query",
	"dot://dns.alidns.com/dns-query",
	"https://dns.digitalsize.net/dns-query",
	"https://v.dnscrypt.uk/dns-query",
	"https://doh.ny.ahadns.net/dns-query",
	"https://ordns.he.net/dns-query",
	"dot://ordns.he.net/dns-query",
	"https://dnscache.e-utp.net/dns-query",
	"dot://dnscache.e-utp.net/dns-query",
	"https://doh.opendns.com/dns-query",
	"https://dns.nextdns.io/dns-query",
	"dot://dns.nextdns.io/dns-query",
	"https://doh.pub/dns-query",
	"dot://doh.pub/dns-query",
	"https://dns1.ipv6.dnscrypt.ca/dns-query",
	"dot://dns1.ipv6.dnscrypt.ca/dns-query",
	"https://dns10.quad9.net/dns-query",
	"dot://dns10.quad9.net/dns-query",
	"https://dns.digitale-gesellschaft.ch/dns-query",
	"dot://dns.digitale-gesellschaft.ch/dns-query",
	"https://dnsforge.de/dns-query",
	"dot://dnsforge.de/dns-query",
	"https://opennic.i2pd.xyz/dns-query",
	"https://doh.nl.ahadns.net/dns-query",
	"https://security.cloudflare-dns.com/dns-query",
	"dot://security.cloudflare-dns.com/dns-query",
	"https://dns2.ipv6.dnscrypt.ca/dns-query",
	"dot://dns2.ipv6.dnscrypt.ca/dns-query",
	"https://dns.therifleman.name/dns-query",
	"dot://dns.therifleman.name/dns-query",
	"https://dns.switch.ch/dns-query",
	"dot://dns.switch.ch/dns-query",
	"https://doh-ch.blahdns.com/dns-query",
	"dot://doh-ch.blahdns.com/dns-query",
	"https://doh.sb/dns-query",
	"dot://doh.sb/dns-query",
	"https://dnsnl-noads.alekberg.net/dns-query",
	"https://doh.tiarap.org/dns-query",
	"https://dns.njal.la/dns-query",
}

var stunReachabilityDefaultInput = stuninput.AsnStunReachabilityInput()

// StaticBareInputForExperiment returns the list of strings an
// experiment should use as static input. In case there is no
// static input for this experiment, we return an error.
func StaticBareInputForExperiment(name string) ([]string, error) {
	// Implementation note: we may be called from pkg/oonimkall
	// with a non-canonical experiment name, so we need to convert
	// the experiment name to be canonical before proceeding.
	switch registry.CanonicalizeExperimentName(name) {
	case "dnscheck":
		return dnsCheckDefaultInput, nil
	case "stunreachability":
		return stunReachabilityDefaultInput, nil
	default:
		return nil, ErrNoStaticInput
	}
}

// staticInputForExperiment returns the static input for the given experiment
// or an error if there's no static input for the experiment.
func staticInputForExperiment(name string) ([]model.OOAPIURLInfo, error) {
	return stringListToModelURLInfo(StaticBareInputForExperiment(name))
}

// loadOrStaticDefault implements the InputOrStaticDefault policy.
func (il *InputLoader) loadOrStaticDefault(ctx context.Context) ([]model.OOAPIURLInfo, error) {
	inputs, err := il.loadLocal()
	if err != nil || len(inputs) > 0 {
		return inputs, err
	}
	return staticInputForExperiment(il.ExperimentName)
}

// loadLocal loads inputs from StaticInputs and SourceFiles.
func (il *InputLoader) loadLocal() ([]model.OOAPIURLInfo, error) {
	inputs := []model.OOAPIURLInfo{}
	for _, input := range il.StaticInputs {
		inputs = append(inputs, model.OOAPIURLInfo{URL: input})
	}
	for _, filepath := range il.SourceFiles {
		extra, err := il.readfile(filepath, fsx.OpenFile)
		if err != nil {
			return nil, err
		}
		// See https://github.com/ooni/probe-engine/issues/1123.
		if len(extra) <= 0 {
			return nil, fmt.Errorf("%w: %s", ErrDetectedEmptyFile, filepath)
		}
		inputs = append(inputs, extra...)
	}
	return inputs, nil
}

// inputLoaderOpenFn is the type of the function to open a file.
type inputLoaderOpenFn func(filepath string) (fs.File, error)

// readfile reads inputs from the specified file. The open argument should be
// compatible with stdlib's fs.Open and helps us with unit testing.
func (il *InputLoader) readfile(filepath string, open inputLoaderOpenFn) ([]model.OOAPIURLInfo, error) {
	inputs := []model.OOAPIURLInfo{}
	filep, err := open(filepath)
	if err != nil {
		return nil, err
	}
	defer filep.Close()
	// Implementation note: when you save file with vim, you have newline at
	// end of file and you don't want to consider that an input line. While there
	// ignore any other empty line that may occur inside the file.
	scanner := bufio.NewScanner(filep)
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			inputs = append(inputs, model.OOAPIURLInfo{URL: line})
		}
	}
	if scanner.Err() != nil {
		return nil, scanner.Err()
	}
	return inputs, nil
}

// loadRemote loads inputs from a remote source.
func (il *InputLoader) loadRemote(ctx context.Context) ([]model.OOAPIURLInfo, error) {
	config := il.CheckInConfig
	if config == nil {
		// Note: Session.CheckIn documentation says it will fill in
		// any field with a required value with a reasonable default
		// if such value is missing. So, here we just need to be
		// concerned about NOT passing it a NULL pointer.
		config = &model.OOAPICheckInConfig{}
	}
	reply, err := il.checkIn(ctx, config)
	if err != nil {
		return nil, err
	}
	if reply.WebConnectivity == nil || len(reply.WebConnectivity.URLs) <= 0 {
		return nil, ErrNoURLsReturned
	}
	return reply.WebConnectivity.URLs, nil
}

// checkIn executes the check-in and filters the returned URLs to exclude
// the URLs that are not part of the requested categories. This is done for
// robustness, just in case we or the API do something wrong.
func (il *InputLoader) checkIn(
	ctx context.Context, config *model.OOAPICheckInConfig) (*model.OOAPICheckInInfo, error) {
	reply, err := il.Session.CheckIn(ctx, config)
	if err != nil {
		return nil, err
	}
	// Note: safe to assume that reply is not nil if err is nil
	if reply.WebConnectivity != nil && len(reply.WebConnectivity.URLs) > 0 {
		reply.WebConnectivity.URLs = il.preventMistakes(
			reply.WebConnectivity.URLs, config.WebConnectivity.CategoryCodes,
		)
	}
	return reply, nil
}

// preventMistakes makes the code more robust with respect to any possible
// integration issue where the backend returns to us URLs that don't
// belong to the category codes we requested.
func (il *InputLoader) preventMistakes(input []model.OOAPIURLInfo, categories []string) (output []model.OOAPIURLInfo) {
	if len(categories) <= 0 {
		return input
	}
	for _, entry := range input {
		var found bool
		for _, cat := range categories {
			if entry.CategoryCode == cat {
				found = true
				break
			}
		}
		if !found {
			il.logger().Warnf("URL %+v not in %+v; skipping", entry, categories)
			continue
		}
		output = append(output, entry)
	}
	return
}

// logger returns the configured logger or apex/log's default.
func (il *InputLoader) logger() InputLoaderLogger {
	if il.Logger != nil {
		return il.Logger
	}
	return log.Log
}

// stringListToModelURLInfo is an utility function to convert
// a list of strings containing URLs into a list of model.URLInfo
// which would have been returned by an hypothetical backend
// API serving input for a test for which we don't have an API
// yet (e.g., stunreachability and dnscheck).
func stringListToModelURLInfo(input []string, err error) ([]model.OOAPIURLInfo, error) {
	if err != nil {
		return nil, err
	}
	var output []model.OOAPIURLInfo
	for _, URL := range input {
		if _, err := url.Parse(URL); err != nil {
			return nil, err
		}
		output = append(output, model.OOAPIURLInfo{
			CategoryCode: "MISC", // hard to find a category
			CountryCode:  "XX",   // representing no country
			URL:          URL,
		})
	}
	return output, nil
}
