// Package targetloading contains common code to load richer-input targets.
package targetloading

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/url"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/experimentname"
	"github.com/ooni/probe-cli/v3/internal/fsx"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/stuninput"
)

// These errors are returned by the [*Loader] or the experiment execution.
var (
	ErrNoURLsReturned    = errors.New("no URLs returned")
	ErrDetectedEmptyFile = errors.New("file did not contain any input")
	ErrInputRequired     = errors.New("no input provided")
	ErrNoInputExpected   = errors.New("we did not expect any input")
	ErrNoStaticInput     = errors.New("no static input for this experiment")
	ErrInvalidInputType  = errors.New("invalid richer input type")
	ErrInvalidInput      = errors.New("input does not conform to spec")
)

// Session is the session according to a [*Loader] instance.
type Session = model.ExperimentTargetLoaderSession

// Logger is the [model.Logger] according to a [*Loader].
type Logger interface {
	// Warnf formats and emits a warning message.
	Warnf(format string, v ...interface{})
}

// Loader loads input according to the specified policy
// either from command line and input files or from OONI services. The
// behaviour depends on the input policy as described below.
//
// You MUST NOT change any public field of this structure when
// in use, because that MAY lead to data races.
//
// # InputNone
//
// We fail if there is any StaticInput or any SourceFiles. If
// there's no input, we return a single, empty entry that causes
// experiments that don't require input to run once.
//
// # InputOptional
//
// We gather input from StaticInput and SourceFiles. If there is
// input, we return it. Otherwise we return a single, empty entry
// that causes experiments that don't require input to run once.
//
// # InputOrQueryBackend
//
// We gather input from StaticInput and SourceFiles. If there is
// input, we return it. Otherwise, we use OONI's probe services
// to gather input using the best API for the task.
//
// # InputOrStaticDefault
//
// We gather input from StaticInput and SourceFiles. If there is
// input, we return it. Otherwise, we return an internal static
// list of inputs to be used with this experiment.
//
// # InputStrictlyRequired
//
// We gather input from StaticInput and SourceFiles. If there is
// input, we return it. Otherwise, we return an error.
type Loader struct {
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

	// Logger is the optional logger that the [*Loader]
	// should be using. If not set, we will use the default
	// logger of github.com/apex/log.
	Logger Logger

	// Session is the current measurement session. You
	// MUST fill in this field.
	Session Session

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
func (il *Loader) Load(ctx context.Context) ([]model.ExperimentTarget, error) {
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
func (il *Loader) loadNone() ([]model.ExperimentTarget, error) {
	if len(il.StaticInputs) > 0 || len(il.SourceFiles) > 0 {
		return nil, ErrNoInputExpected
	}
	// Implementation note: the convention for input-less experiments is that
	// they require a single entry containing an empty input.
	entry := model.NewOOAPIURLInfoWithDefaultCategoryAndCountry("")
	return []model.ExperimentTarget{entry}, nil
}

// loadOptional implements the InputOptional policy.
func (il *Loader) loadOptional() ([]model.ExperimentTarget, error) {
	inputs, err := il.loadLocal()
	if err == nil && len(inputs) <= 0 {
		// Implementation note: the convention for input-less experiments is that
		// they require a single entry containing an empty input.
		inputs = []model.ExperimentTarget{model.NewOOAPIURLInfoWithDefaultCategoryAndCountry("")}
	}
	return inputs, err
}

// loadStrictlyRequired implements the InputStrictlyRequired policy.
func (il *Loader) loadStrictlyRequired(_ context.Context) ([]model.ExperimentTarget, error) {
	inputs, err := il.loadLocal()
	if err != nil || len(inputs) > 0 {
		return inputs, err
	}
	return nil, ErrInputRequired
}

// loadOrQueryBackend implements the InputOrQueryBackend policy.
func (il *Loader) loadOrQueryBackend(ctx context.Context) ([]model.ExperimentTarget, error) {
	inputs, err := il.loadLocal()
	if err != nil || len(inputs) > 0 {
		return inputs, err
	}
	// This assumes that the default experiment for loading remote targets is WebConnectivity.
	return il.loadRemoteWebConnectivity(ctx)
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
	"https://family.cloudflare-dns.com/dns-query",
	"dot://family.cloudflare-dns.com/dns-query",
	"https://dns11.quad9.net/dns-query",
	"dot://dns11.quad9.net/dns-query",
	"https://dns9.quad9.net/dns-query",
	"dot://dns9.quad9.net/dns-query",
	"https://dns12.quad9.net/dns-query",
	"dot://dns12.quad9.net/dns-query",
	"https://1dot1dot1dot1.cloudflare-dns.com/dns-query",
	"dot://1dot1dot1dot1.cloudflare-dns.com/dns-query",
	"https://dns.adguard.com/dns-query",
	"dot://dns.adguard.com/dns-query",
	"https://dns-family.adguard.com/dns-query",
	"dot://dns-family.adguard.com/dns-query",
	"https://dns.cloudflare.com/dns-query",
	"https://adblock.doh.mullvad.net/dns-query",
	"dot://adblock.doh.mullvad.net/dns-query",
	"https://dns.alidns.com/dns-query",
	"dot://dns.alidns.com/dns-query",
	"https://doh.opendns.com/dns-query",
	"https://dns.nextdns.io/dns-query",
	"dot://dns.nextdns.io/dns-query",
	"https://dns10.quad9.net/dns-query",
	"dot://dns10.quad9.net/dns-query",
	"https://security.cloudflare-dns.com/dns-query",
	"dot://security.cloudflare-dns.com/dns-query",
	"https://dns.switch.ch/dns-query",
	"dot://dns.switch.ch/dns-query",
}

var stunReachabilityDefaultInput = stuninput.AsnStunReachabilityInput()

// staticBareInputForExperiment returns the list of strings an
// experiment should use as static input. In case there is no
// static input for this experiment, we return an error.
func staticBareInputForExperiment(name string) ([]string, error) {
	// Implementation note: we may be called from pkg/oonimkall
	// with a non-canonical experiment name, so we need to convert
	// the experiment name to be canonical before proceeding.
	switch experimentname.Canonicalize(name) {
	case "dnscheck":
		// TODO(https://github.com/ooni/probe/issues/1390): serve DNSCheck
		// inputs using richer input (aka check-in v2).
		return dnsCheckDefaultInput, nil
	case "stunreachability":
		// TODO(https://github.com/ooni/probe/issues/2557): server STUNReachability
		// inputs using richer input (aka check-in v2).
		return stunReachabilityDefaultInput, nil
	default:
		return nil, ErrNoStaticInput
	}
}

// staticInputForExperiment returns the static input for the given experiment
// or an error if there's no static input for the experiment.
func staticInputForExperiment(name string) ([]model.ExperimentTarget, error) {
	return stringListToModelExperimentTarget(staticBareInputForExperiment(name))
}

// loadOrStaticDefault implements the InputOrStaticDefault policy.
func (il *Loader) loadOrStaticDefault(_ context.Context) ([]model.ExperimentTarget, error) {
	inputs, err := il.loadLocal()
	if err != nil || len(inputs) > 0 {
		return inputs, err
	}
	return staticInputForExperiment(il.ExperimentName)
}

// loadLocal loads inputs from the [*Loader] StaticInputs and SourceFiles.
func (il *Loader) loadLocal() ([]model.ExperimentTarget, error) {
	inputs, err := LoadStatic(il)
	if err != nil {
		return nil, err
	}
	var targets []model.ExperimentTarget
	for _, input := range inputs {
		targets = append(targets, model.NewOOAPIURLInfoWithDefaultCategoryAndCountry(input))
	}
	return targets, nil
}

// openFunc is the type of the function to open a file.
type openFunc func(filepath string) (fs.File, error)

// readfile reads inputs from the specified file. The open argument should be
// compatible with stdlib's fs.Open and helps us with unit testing.
func readfile(filepath string, open openFunc) ([]string, error) {
	inputs := []string{}
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
			inputs = append(inputs, line)
		}
	}
	if scanner.Err() != nil {
		return nil, scanner.Err()
	}
	return inputs, nil
}

// LoadStatic loads inputs from the [*Loader] StaticInputs and SourceFiles.
func LoadStatic(config *Loader) ([]string, error) {
	inputs := append([]string{}, config.StaticInputs...)
	for _, filepath := range config.SourceFiles {
		extra, err := readfile(filepath, fsx.OpenFile)
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

// loadRemoteWebConnectivity loads webconnectivity inputs from a remote source.
func (il *Loader) loadRemoteWebConnectivity(ctx context.Context) ([]model.ExperimentTarget, error) {
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
	output := modelOOAPIURLInfoToModelExperimentTarget(reply.WebConnectivity.URLs)
	return output, nil
}

func modelOOAPIURLInfoToModelExperimentTarget(
	inputs []model.OOAPIURLInfo) (outputs []model.ExperimentTarget) {
	for _, input := range inputs {
		// Note: Dammit! Before we switch to go1.22 we need to continue to
		// stay careful about the variable over which we're looping!
		//
		// See https://go.dev/blog/loopvar-preview for more information.
		outputs = append(outputs, &model.OOAPIURLInfo{
			CategoryCode: input.CategoryCode,
			CountryCode:  input.CountryCode,
			URL:          input.URL,
		})
	}
	return
}

// checkIn executes the check-in and filters the returned URLs to exclude
// the URLs that are not part of the requested categories. This is done for
// robustness, just in case we or the API do something wrong.
func (il *Loader) checkIn(
	ctx context.Context, config *model.OOAPICheckInConfig) (*model.OOAPICheckInResultNettests, error) {
	reply, err := il.Session.CheckIn(ctx, config)
	if err != nil {
		return nil, err
	}
	// Note: safe to assume that reply is not nil if err is nil
	if reply.Tests.WebConnectivity != nil && len(reply.Tests.WebConnectivity.URLs) > 0 {
		reply.Tests.WebConnectivity.URLs = il.preventMistakes(
			reply.Tests.WebConnectivity.URLs, config.WebConnectivity.CategoryCodes,
		)
	}
	return &reply.Tests, nil
}

// preventMistakes makes the code more robust with respect to any possible
// integration issue where the backend returns to us URLs that don't
// belong to the category codes we requested.
func (il *Loader) preventMistakes(input []model.OOAPIURLInfo, categories []string) (output []model.OOAPIURLInfo) {
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
func (il *Loader) logger() Logger {
	if il.Logger != nil {
		return il.Logger
	}
	return log.Log
}

// stringListToModelExperimentTarget is an utility function to convert
// a list of strings containing URLs into a list of model.ExperimentTarget
// which would have been returned by an hypothetical backend
// API serving input for a test for which we don't have an API
// yet (e.g., stunreachability and dnscheck).
func stringListToModelExperimentTarget(input []string, err error) ([]model.ExperimentTarget, error) {
	if err != nil {
		return nil, err
	}
	var output []model.ExperimentTarget
	for _, URL := range input {
		if _, err := url.Parse(URL); err != nil {
			return nil, err
		}
		output = append(output, model.NewOOAPIURLInfoWithDefaultCategoryAndCountry(URL))
	}
	return output, nil
}
