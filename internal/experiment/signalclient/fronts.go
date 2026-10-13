package signalclient

//
// The domain fronts of the circumvention routes, from libsignal v0.105.0 rust/net/src/env.rs and
// Signal-Android v8.31.1 SignalServiceNetworkAccess.kt (ts-041 Appendix B), and the hello each
// route's client sends to them. The Fronts option replaces a route's fronts.
//

import (
	"crypto/rand"
	"math/big"
)

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

// googleMapsFronts are the Google fronts Signal-Android reaches with GMAPS_CONNECTION_SPEC; its
// other Google fronts (www.google.com, googlemail.com, android.clients.google.com and the
// countries' domains) use GMAIL_CONNECTION_SPEC or PLAY_CONNECTION_SPEC, the same list.
var googleMapsFronts = map[string]bool{"clients3.google.com": true, "clients4.google.com": true}

// frontHello is the hello a route's client sends to front for service ("chat" or "storage").
// A front given with the Fronts option that Signal-Android does not use gets the hello of the
// route's default fronts (Gmail's on android_g).
func frontHello(route, service, front string) Hello {
	switch {
	case route == RouteReflectorF || route == RouteReflectorG:
		return HelloLibsignalReflector
	case route == RouteAndroidF && service == "chat":
		return HelloAndroidModernWebSocket
	case route == RouteAndroidF:
		return HelloAndroidModernREST
	case googleMapsFronts[front] && service == "chat":
		return HelloAndroidGmapsWebSocket
	case googleMapsFronts[front]:
		return HelloAndroidGmapsREST
	case service == "chat":
		return HelloAndroidGmailWebSocket
	default:
		return HelloAndroidGmailREST
	}
}
