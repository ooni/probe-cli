// Package stuninput contains stun targets as well as
// code to format such targets according to various conventions.
package stuninput

import (
	"fmt"
	"net/url"
)

// TODO(bassosimone): we need to keep this list in sync with
// the list internally used by TPO's snowflake.
var inputs = []string{
	"stun.voip.blackberry.com:3478",
	"stun.antisip.com:3478",
	"stun.bluesip.net:3478",
	"stun.dus.net:3478",
	"stun.epygi.com:3478",
	"stun.sonetel.com:3478",
	"stun.sonetel.net:3478",
	"stun.uls.co.za:3478",
	"stun.voipgate.com:3478",
	"stun.voys.nl:3478",
}

// AsSnowflakeInput formats the input in the format
// that is expected by snowflake.
func AsSnowflakeInput() (output []string) {
	for _, input := range inputs {
		output = append(output, fmt.Sprintf("stun:%s", input))
	}
	return
}

// AsnStunReachabilityInput formats the input in
// the format that is expected by stunreachability.
func AsnStunReachabilityInput() (output []string) {
	for _, input := range inputs {
		serio := (&url.URL{Scheme: "stun", Host: input})
		output = append(output, serio.String())
	}
	return
}
