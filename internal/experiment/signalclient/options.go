package signalclient

//
// Options that take lists (Controls, StaticAddresses, Circumvention, Fronts) are strings, since
// OONI options are scalars. They are parsed once, before the measurement starts: an invalid
// option fails the run instead of measuring something other than what was asked.
//

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
)

// Values of the Controls option.
const (
	ControlsOnFailure = "on_failure"
	ControlsAlways    = "always"
	ControlsOff       = "off"
)

// Circumvention route names.
const (
	RouteReflectorF = "reflector_f"
	RouteReflectorG = "reflector_g"
	RouteAndroidG   = "android_g"
	RouteAndroidF   = "android_f"
)

// allRoutes is every route, in the order they run.
var allRoutes = []string{RouteReflectorF, RouteReflectorG, RouteAndroidG, RouteAndroidF}

// frontConfig is how one route (or one service of an Android route) reaches Signal through a CDN.
type frontConfig struct {
	hostHeader string
	snis       []string
	pathPrefix string // the Android Google front's per-service prefix ("/service", "/storage")
}

// Fronts, from libsignal v0.105.0 rust/net/src/env.rs and Signal-Android v8.31.1
// SignalServiceNetworkAccess.kt (ts-041 Appendix B).
var (
	fastlyFronts = []string{"github.githubassets.com", "pinterest.com", "www.redditstatic.com"}
	googleFronts = []string{"www.google.com", "android.clients.google.com", "clients3.google.com",
		"clients4.google.com", "googlemail.com"}

	// googleCountryFronts are the countries for which Signal-Android uses the Google front,
	// with their own Google domain, which comes first.
	googleCountryFronts = map[string]string{
		"EG": "www.google.com.eg", "AE": "www.google.ae", "OM": "www.google.com.om",
		"QA": "www.google.com.qa", "UZ": "www.google.co.uz", "VE": "www.google.co.ve",
		"PK": "www.google.com.pk",
	}

	// fastlyCountries are the countries for which Signal-Android uses the Fastly front.
	fastlyCountries = map[string]bool{"IR": true, "CU": true}
)

const googleReflectorHost = "reflector-nrgwuv7kwq-uc.a.run.app"

// builtinFronts returns the front configuration for country, keyed by route or route.service.
func builtinFronts(country string) map[string]frontConfig {
	android := googleFronts
	if domain, ok := googleCountryFronts[country]; ok {
		android = append([]string{domain}, googleFronts...)
	}
	return map[string]frontConfig{
		RouteReflectorF:            {hostHeader: "reflector-signal.global.ssl.fastly.net", snis: fastlyFronts},
		RouteReflectorG:            {hostHeader: googleReflectorHost, snis: googleFronts},
		RouteAndroidG + ".chat":    {hostHeader: googleReflectorHost, snis: android, pathPrefix: "/service"},
		RouteAndroidG + ".storage": {hostHeader: googleReflectorHost, snis: android, pathPrefix: "/storage"},
		RouteAndroidF + ".chat":    {hostHeader: "chat-signal.global.ssl.fastly.net", snis: fastlyFronts},
		RouteAndroidF + ".storage": {hostHeader: "storage.signal.org.global.prod.fastly.net", snis: fastlyFronts},
	}
}

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

// pickFront chooses the SNI a route uses: at random among the reflectors' fronts, as libsignal
// does, and among the Fastly fronts for the countries Signal-Android sends there; otherwise the
// first front, which for the Google front is the country's own domain when it has one.
// Signal-Android itself picks at random among its fronts each time (chooseUrl); the fixed choice
// keeps one network's measurements comparable across runs.
func pickFront(route, country string, snis []string) string {
	random := route == RouteReflectorF || route == RouteReflectorG || (route == RouteAndroidF && fastlyCountries[country])
	if !random || len(snis) < 2 {
		return snis[0]
	}
	n, err := rand.Int(rand.Reader, big.NewInt(int64(len(snis))))
	if err != nil {
		return snis[0]
	}
	return snis[n.Int64()]
}
