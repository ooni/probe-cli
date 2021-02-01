package errorx

import "regexp"

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

const ipv4Address = `\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}`
const ipv6Address = `([0-9a-fA-F]{0,4}:){5,7}([0-9a-fA-F]{0,4})?`
const ipv6Compressed = `([0-9a-fA-F]{0,4}:){0,5}([0-9a-fA-F]{0,4})?(::)([0-9a-fA-F]{0,4}:){0,5}([0-9a-fA-F]{0,4})?`
const ipv6Full = `(` + ipv6Address + `(` + ipv4Address + `))` +
	`|(` + ipv6Compressed + `(` + ipv4Address + `))` +
	`|(` + ipv6Address + `)` + `|(` + ipv6Compressed + `)`
const optionalPort = `(:\d{1,5})?`
const addressPattern = `((` + ipv4Address + `)|(\[(` + ipv6Full + `)\])|(` + ipv6Full + `))` + optionalPort
const fullAddrPattern = `(^|\s|[^\w:])` + addressPattern + `(\s|(:\s)|[^\w:]|$)`

var scrubberPatterns = []*regexp.Regexp{
	regexp.MustCompile(fullAddrPattern),
}

var addressRegexp = regexp.MustCompile(addressPattern)

func scrub(b []byte) []byte {
	scrubbedBytes := b
	for _, pattern := range scrubberPatterns {
		// this is a workaround since go does not yet support look ahead or look
		// behind for regular expressions.
		scrubbedBytes = pattern.ReplaceAllFunc(scrubbedBytes, func(b []byte) []byte {
			return addressRegexp.ReplaceAll(b, []byte("[scrubbed]"))
		})
	}
	return scrubbedBytes
}

// Scrub sanitizes a string containing an error such that
// any occurrence of IP endpoints is scrubbed
func Scrub(s string) string {
	return string(scrub([]byte(s)))
}
