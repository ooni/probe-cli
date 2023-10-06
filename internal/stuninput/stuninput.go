// Package stuninput contains stun targets as well as
// code to format such targets according to various conventions.
package stuninput

import (
	"fmt"
	"net/url"
)

// TODO(bassosimone): we need to keep this list in sync with
// the list internally used by TPO's snowflake.
//
// We should sync with https://gitlab.torproject.org/tpo/applications/tor-browser-build/-/blob/main/projects/tor-expert-bundle/pt_config.json
var inputs = map[string]bool{
	"stun.l.google.com:19302": true,
	"stun.antisip.com:3478":   true,
	"stun.bluesip.net:3478":   true,
	"stun.dus.net:3478":       true,
	"stun.epygi.com:3478":     true,
	"stun.sonetel.com:3478":   true,
	"stun.uls.co.za:3478":     true,
	"stun.voipgate.com:3478":  true,
	"stun.voys.nl:3478":       true,
}

// AsSnowflakeInput formats the input in the format
// that is expected by snowflake.
func AsSnowflakeInput() (output []string) {
	for input := range inputs {
		output = append(output, fmt.Sprintf("stun:%s", input))
	}
	return
}

// AsnStunReachabilityInput formats the input in
// the format that is expected by stunreachability.
func AsnStunReachabilityInput() (output []string) {
	for input := range inputs {
		serio := (&url.URL{Scheme: "stun", Host: input})
		output = append(output, serio.String())
	}
	return
}
