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
