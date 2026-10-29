// Package browserhttp fetches web pages the way a browser does.
//
// Bot detection compares the TLS ClientHello, the HTTP/2 framing, and the
// headers' order, case and values with what the claimed browser sends, and
// refuses clients that don't match. Web Connectivity and its test helper use
// this package so that both present the browser's fingerprint.
package browserhttp

import (
	"net/http"

	"github.com/ooni/probe-cli/v3/internal/model"
	utls "github.com/refraction-networking/utls"
)

// We claim to be the Chrome on stable (156 on 2026-10-29). Update the
// ClientHello, the User-Agent and sec-ch-ua together, to a current Chrome,
// and the reference values in browserhttptest from a capture of it.

// ClientHello is uTLS's newest Chrome ClientHello, captured from Chrome 155.
var ClientHello = &utls.HelloChrome_155

// UserAgent is Chrome 156's on Windows. It doesn't follow
// [model.HTTPHeaderUserAgent], which other experiments use.
const UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/156.0.0.0 Safari/537.36"

// SecChUA is Chrome 156's sec-ch-ua header. Chromium derives the GREASE
// brand and the order from the major version.
const SecChUA = `"Not:A-Brand";v="8", "Chromium";v="156", "Google Chrome";v="156"`

// Accept is Chrome 156's Accept header for navigations.
const Accept = "text/html,application/xhtml+xml,application/xml;q=0.9,image/jxl,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7"

// SetHeaders sets the headers Chrome sends when the user opens a URL.
// Chrome sends the sec-ch-ua and Sec-Fetch headers, and offers brotli and
// zstd, only over HTTPS. [Transport] writes them in Chrome's order and case,
// and decodes the body.
func SetHeaders(header http.Header, secure bool) {
	if secure {
		header.Set("sec-ch-ua", SecChUA)
		header.Set("sec-ch-ua-mobile", "?0")
		header.Set("sec-ch-ua-platform", `"Windows"`)
	}
	header.Set("Upgrade-Insecure-Requests", "1")
	header.Set("User-Agent", UserAgent)
	header.Set("Accept", Accept)
	if secure {
		header.Set("Sec-Fetch-Site", "none")
		header.Set("Sec-Fetch-Mode", "navigate")
		header.Set("Sec-Fetch-User", "?1")
		header.Set("Sec-Fetch-Dest", "document")
		header.Set("Accept-Encoding", "gzip, deflate, br, zstd")
	} else {
		for _, name := range []string{"sec-ch-ua", "sec-ch-ua-mobile", "sec-ch-ua-platform",
			"Sec-Fetch-Site", "Sec-Fetch-Mode", "Sec-Fetch-User", "Sec-Fetch-Dest"} {
			header.Del(name)
		}
		header.Set("Accept-Encoding", "gzip, deflate")
	}
	header.Set("Accept-Language", model.HTTPHeaderAcceptLanguage)
}

// IsBrowserRequest tells whether headers come from [SetHeaders]: a probe
// that fetches like a browser asks its test helper to do the same.
func IsBrowserRequest(header http.Header) bool {
	return header.Get("Upgrade-Insecure-Requests") == "1"
}

// HeaderOrder is the order and case in which Chrome writes a navigation's
// headers over HTTP/1.1, after Host and Connection. Over HTTP/2 it uses
// the same order, lowercase, and adds priority last.
var HeaderOrder = []string{
	"sec-ch-ua",
	"sec-ch-ua-mobile",
	"sec-ch-ua-platform",
	"Upgrade-Insecure-Requests",
	"User-Agent",
	"Accept",
	"Sec-Fetch-Site",
	"Sec-Fetch-Mode",
	"Sec-Fetch-User",
	"Sec-Fetch-Dest",
	"Referer",
	"Accept-Encoding",
	"Accept-Language",
	"Cookie",
}
