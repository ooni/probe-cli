package signalclient

//
// Options that take lists (Controls, StaticAddresses, Circumvention, Fronts) are strings, since
// OONI options are scalars. They are parsed once, before the measurement starts: an invalid
// option fails the run instead of measuring something other than what was asked.
//

import (
	"fmt"
	"strings"
)

// Config contains the experiment config.
type Config struct {
	// IdleSeconds keeps each provisioning socket idle this long before the keepalives, to detect
	// middleboxes that kill idle sessions. It is capped at maxIdleSeconds.
	IdleSeconds int64 `ooni:"seconds of idle time on provisioning sockets before keepalives (0 = none)"`

	// AppUserAgent sends each real client's User-Agent instead of identifying as OONI Probe.
	// By default the test identifies itself (see ooniUserAgent), so that Signal can recognize
	// and filter its traffic. The User-Agent travels inside TLS: either way, the network sees
	// the same bytes.
	AppUserAgent bool `ooni:"send the Signal apps' User-Agents instead of identifying as OONI Probe"`

	// UserAgent, when set, is sent instead of either.
	UserAgent string `ooni:"User-Agent to send instead of the default"`

	// SignalCA, when set, is a PEM bundle that replaces Signal's two built-in roots for every
	// connection verified against Signal's roots (targets, controls, inner connections of the
	// circumvention routes), for a root rotation that ships before a probe release or a private
	// Signal server. Not for the fronts or DNS-over-HTTPS.
	SignalCA string `ooni:"PEM bundle replacing Signal's two built-in roots"`

	// Controls says when the address controls run: on_failure (default), always or off.
	Controls string `ooni:"when the address controls run: on_failure (default), always or off"`

	// StaticAddresses replaces the built-in static addresses of the named hosts:
	// "host=addr,addr;host=addr".
	StaticAddresses string `ooni:"replace static addresses: host=addr,addr;host=addr"`

	// Circumvention lists the circumvention routes to measure: a comma-separated subset of
	// reflector_f, reflector_g, android_g, android_f, or all. Empty: none.
	Circumvention string `ooni:"circumvention routes: comma-separated reflector_f,reflector_g,android_g,android_f or all"`

	// CircumventionCountry is the country whose Android front configuration android_g and
	// android_f reproduce (default: the probe's country).
	CircumventionCountry string `ooni:"country of the Android front configuration (default: the probe's)"`

	// Fronts replaces the built-in front configuration of the named routes:
	// "route[.service]=host_header@sni,sni;...".
	Fronts string `ooni:"replace fronts: route[.service]=host_header@sni,sni;..."`
}

// ooniUserAgent identifies this test to Signal's servers.
const ooniUserAgent = "ooniprobe-" + testName + "/" + testVersion + " (+https://ooni.org)"

// userAgent is the User-Agent to send with hello.
func (c *Config) userAgent(hello Hello) string {
	switch {
	case c.UserAgent != "":
		return c.UserAgent
	case c.AppUserAgent && hello.appUserAgent != "":
		return hello.appUserAgent
	case hello.libsignal: // libsignal appends its own token, as it does for the apps
		return ooniUserAgent + " libsignal/" + libsignalVersion
	default:
		return ooniUserAgent
	}
}

// runOptions are the parsed options of one run.
type runOptions struct {
	controls string
	static   map[string][]staticAddress
	routes   []string
	country  string
	fronts   map[string]frontConfig
}

// parseOptions parses and validates the options. probeCC is the default CircumventionCountry.
func (m *Measurer) parseOptions(probeCC string) (*runOptions, error) {
	controls, err := parseControls(m.config.Controls)
	if err != nil {
		return nil, err
	}
	static := builtinStaticAddresses()
	override, err := parseStaticAddresses(m.config.StaticAddresses)
	if err != nil {
		return nil, err
	}
	for host, list := range override {
		static[host] = list
	}
	routes, err := parseCircumvention(m.config.Circumvention)
	if err != nil {
		return nil, err
	}
	country := strings.ToUpper(strings.TrimSpace(m.config.CircumventionCountry))
	if country == "" {
		country = strings.ToUpper(probeCC)
	}
	fronts := builtinFronts(country)
	if err := parseFronts(m.config.Fronts, fronts); err != nil {
		return nil, err
	}
	return &runOptions{controls: controls, static: static, routes: routes, country: country, fronts: fronts}, nil
}

// Values of the Controls option.
const (
	ControlsOnFailure = "on_failure"
	ControlsAlways    = "always"
	ControlsOff       = "off"
)

// parseControls validates the Controls option.
func parseControls(option string) (string, error) {
	switch option {
	case "":
		return ControlsOnFailure, nil
	case ControlsOnFailure, ControlsAlways, ControlsOff:
		return option, nil
	}
	return "", fmt.Errorf("signal_client: Controls: %q is not on_failure, always or off", option)
}

// parseCircumvention parses the Circumvention option into routes, in the order they run.
func parseCircumvention(option string) ([]string, error) {
	want := map[string]bool{}
	for _, name := range strings.Split(option, ",") {
		switch name = strings.TrimSpace(name); name {
		case "":
		case "all":
			for _, r := range allRoutes {
				want[r] = true
			}
		case RouteReflectorF, RouteReflectorG, RouteAndroidG, RouteAndroidF:
			want[name] = true
		default:
			return nil, fmt.Errorf("signal_client: Circumvention: unknown route %q", name)
		}
	}
	var out []string
	for _, r := range allRoutes {
		if want[r] {
			out = append(out, r)
		}
	}
	return out, nil
}

// parseFronts applies the Fronts option, "route[.service]=host_header@sni,sni;...", to fronts.
// An Android route without a service sets both of its services.
func parseFronts(option string, fronts map[string]frontConfig) error {
	for _, entry := range strings.Split(option, ";") {
		if entry = strings.TrimSpace(entry); entry == "" {
			continue
		}
		key, value, ok := strings.Cut(entry, "=")
		key = strings.TrimSpace(key)
		host, snis, ok2 := strings.Cut(value, "@")
		host = strings.TrimSpace(host)
		if !ok || !ok2 || host == "" {
			return fmt.Errorf("signal_client: Fronts: invalid entry %q", entry)
		}
		var list []string
		for _, s := range strings.Split(snis, ",") {
			if s = strings.TrimSpace(s); s != "" {
				list = append(list, s)
			}
		}
		if len(list) == 0 {
			return fmt.Errorf("signal_client: Fronts: no SNI in %q", entry)
		}
		var keys []string
		switch key {
		case RouteReflectorF, RouteReflectorG,
			RouteAndroidG + ".chat", RouteAndroidG + ".storage", RouteAndroidF + ".chat", RouteAndroidF + ".storage":
			keys = []string{key}
		case RouteAndroidG, RouteAndroidF:
			keys = []string{key + ".chat", key + ".storage"}
		default:
			return fmt.Errorf("signal_client: Fronts: unknown route %q", key)
		}
		for _, k := range keys {
			fc := fronts[k]
			fc.hostHeader, fc.snis = host, list
			fronts[k] = fc
		}
	}
	return nil
}
