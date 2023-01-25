package scrubber

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

// The code in this file is adapted from github.com/keroserene/snowflake's
// common/safelog/safelog.go implementation <https://git.io/JfO9w>.
//
// ================================================================================
// Copyright (c) 2016, Serene Han, Arlo Breault
// Copyright (c) 2019-2020, The Tor Project, Inc
//
// Redistribution and use in source and binary forms, with or without modification,
// are permitted provided that the following conditions are met:
//
//   * Redistributions of source code must retain the above copyright notice, this
// list of conditions and the following disclaimer.
//
//   * Redistributions in binary form must reproduce the above copyright notice,
// this list of conditions and the following disclaimer in the documentation and/or
// other materials provided with the distribution.
//
//   * Neither the names of the copyright owners nor the names of its
// contributors may be used to endorse or promote products derived from this
// software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
// ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
// WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE FOR
// ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON
// ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
// ================================================================================

// Test the log scrubber on known problematic log messages
func TestLogScrubberMessages(t *testing.T) {
	for _, test := range []struct {
		input, expected string
	}{
		{
			"http: TLS handshake error from 129.97.208.23:38310: ",
			"http: TLS handshake error from [scrubbed]: ",
		},
		{
			"http2: panic serving [2620:101:f000:780:9097:75b1:519f:dbb8]:58344: interface conversion: *http2.responseWriter is not http.Hijacker: missing method Hijack",
			"http2: panic serving [scrubbed]: interface conversion: *http2.responseWriter is not http.Hijacker: missing method Hijack",
		},
		{
			//Make sure it doesn't scrub fingerprint
			"a=fingerprint:sha-256 33:B6:FA:F6:94:CA:74:61:45:4A:D2:1F:2C:2F:75:8A:D9:EB:23:34:B2:30:E9:1B:2A:A6:A9:E0:44:72:CC:74",
			"a=fingerprint:sha-256 33:B6:FA:F6:94:CA:74:61:45:4A:D2:1F:2C:2F:75:8A:D9:EB:23:34:B2:30:E9:1B:2A:A6:A9:E0:44:72:CC:74",
		},
		{
			//try with enclosing parens
			"(1:2:3:4:c:d:e:f) {1:2:3:4:c:d:e:f}",
			"([scrubbed]) {[scrubbed]}",
		},
		{
			//Make sure it doesn't scrub timestamps
			"2019/05/08 15:37:31 starting",
			"2019/05/08 15:37:31 starting",
		},
		{
			//Make sure ipv6 addresses where : are encoded as %3A or %3a are scrubbed
			"error dialing relay: wss://snowflake.torproject.net/?client_ip=6201%3ac8%3A3004%3A%3A1234",
			"error dialing relay: wss://snowflake.torproject.net/?client_ip=[scrubbed]",
		},
		{
			// make sure url encoded IPv6 IPs get scrubbed (%3a)
			"http2: panic serving [fd00%3a111%3af000%3a777%3a9999%3abbbb%3affff%3adddd]:58344: xxx",
			"http2: panic serving [scrubbed]: xxx",
		},
		{
			// make sure url encoded IPv6 IPs get scrubbed (%3A)
			"http2: panic serving [fd00%3a111%3af000%3a777%3a9999%3abbbb%3affff%3adddd]:58344: xxx",
			"http2: panic serving [scrubbed]: xxx",
		},
		{
			// make sure url encoded IPv6 IPs get scrubbed, different URL (%3A)
			"error dialing relay: wss://snowflake.torproject.net/?client_ip=fd00%3A8888%3Abbbb%3Acccc%3Adddd%3Aeeee%3A2222%3A123 = dial tcp xxx",
			"error dialing relay: wss://snowflake.torproject.net/?client_ip=[scrubbed] = dial tcp xxx",
		},
		{
			// make sure url encoded IPv6 IPs get scrubbed (%3A), compressed
			"http2: panic serving [1%3A2%3A3%3A%3Ad%3Ae%3Af]:55: xxx",
			"http2: panic serving [scrubbed]: xxx",
		},
		{
			// make sure url encoded IPv6 IPs get scrubbed (%3A), compressed
			"error dialing relay: wss://snowflake.torproject.net/?client_ip=1%3A2%3A3%3A%3Ad%3Ae%3Af = dial tcp xxx",
			"error dialing relay: wss://snowflake.torproject.net/?client_ip=[scrubbed] = dial tcp xxx",
		},
	} {
		if Scrub(test.input) != test.expected {
			t.Error(cmp.Diff(test.input, test.expected))
		}
	}
}

func TestLogScrubberGoodFormats(t *testing.T) {
	for _, addr := range []string{
		// IPv4
		"1.2.3.4",
		"255.255.255.255",
		// IPv4 with port
		"1.2.3.4:55",
		"255.255.255.255:65535",
		// IPv6
		"1:2:3:4:c:d:e:f",
		"1111:2222:3333:4444:CCCC:DDDD:EEEE:FFFF",
		// IPv6 with brackets
		"[1:2:3:4:c:d:e:f]",
		"[1111:2222:3333:4444:CCCC:DDDD:EEEE:FFFF]",
		// IPv6 with brackets and port
		"[1:2:3:4:c:d:e:f]:55",
		"[1111:2222:3333:4444:CCCC:DDDD:EEEE:FFFF]:65535",
		// compressed IPv6
		"::f",
		"::d:e:f",
		"1:2:3::",
		"1:2:3::d:e:f",
		"1:2:3:d:e:f::",
		"::1:2:3:d:e:f",
		"1111:2222:3333::DDDD:EEEE:FFFF",
		// compressed IPv6 with brackets
		"[::d:e:f]",
		"[1:2:3::]",
		"[1:2:3::d:e:f]",
		"[1111:2222:3333::DDDD:EEEE:FFFF]",
		"[1:2:3:4:5:6::8]",
		"[1::7:8]",
		// compressed IPv6 with brackets and port
		"[1::]:58344",
		"[::d:e:f]:55",
		"[1:2:3::]:55",
		"[1:2:3::d:e:f]:55",
		"[1111:2222:3333::DDDD:EEEE:FFFF]:65535",
		// IPv4-compatible and IPv4-mapped
		"::255.255.255.255",
		"::ffff:255.255.255.255",
		"[::255.255.255.255]",
		"[::ffff:255.255.255.255]",
		"[::255.255.255.255]:65535",
		"[::ffff:255.255.255.255]:65535",
		"[::ffff:0:255.255.255.255]",
		"[2001:db8:3:4::192.0.2.33]",
	} {
		if Scrub(addr) != "[scrubbed]" {
			t.Error(cmp.Diff(addr, "[scrubbed]"))
		}
	}
}
