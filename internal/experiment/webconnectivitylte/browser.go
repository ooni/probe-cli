package webconnectivitylte

//
// The browser our HTTP(S) fetches look like
//
// Bot detection compares the TLS ClientHello, the headers and the User-Agent
// with what the claimed browser sends, and refuses probes that don't match.
//

import (
	"net/http"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// We claim to be the Chrome on stable (156 on 2026-10-29). Update the
// User-Agent and sec-ch-ua together, to a current Chrome.

// browserUserAgent is Chrome 156's on Windows. It doesn't follow
// [model.HTTPHeaderUserAgent], which other experiments use.
const browserUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/156.0.0.0 Safari/537.36"

// browserSecChUA is Chrome 156's sec-ch-ua header. Chromium derives the
// GREASE brand and the order from the major version.
const browserSecChUA = `"Not:A-Brand";v="8", "Chromium";v="156", "Google Chrome";v="156"`

// browserAccept is Chrome's Accept header for navigations.
const browserAccept = "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7"

// browserSetHeaders sets the headers Chrome sends when the user opens
// a URL. Chrome sends the sec-ch-ua and Sec-Fetch headers only over HTTPS.
func browserSetHeaders(header http.Header, secure bool) {
	if secure {
		// Chrome sends these names in lowercase, also over HTTP/1.1
		header["sec-ch-ua"] = []string{browserSecChUA}
		header["sec-ch-ua-mobile"] = []string{"?0"}
		header["sec-ch-ua-platform"] = []string{`"Windows"`}
	}
	header.Set("Upgrade-Insecure-Requests", "1")
	header.Set("User-Agent", browserUserAgent)
	header.Set("Accept", browserAccept)
	if secure {
		header.Set("Sec-Fetch-Site", "none")
		header.Set("Sec-Fetch-Mode", "navigate")
		header.Set("Sec-Fetch-User", "?1")
		header.Set("Sec-Fetch-Dest", "document")
	}
	header.Set("Accept-Language", model.HTTPHeaderAcceptLanguage)
}
