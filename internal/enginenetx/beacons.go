package enginenetx

//
// beacons policy - a policy where we treat some IP addresses as special for
// some domains, bypassing DNS lookups and using custom SNIs
//

import (
	"context"
	"math/rand"
	"time"
)

// beaconsPolicy is a policy where we use beacons for communicating
// with the OONI backend, i.e., api.ooni.io.
//
// A beacon is an IP address that can route traffic from and to
// the OONI backend and accepts any SNI.
//
// The zero value is invalid; please, init MANDATORY fields.
type beaconsPolicy struct {
	// Fallback is the MANDATORY fallback policy.
	Fallback HTTPSDialerPolicy
}

var _ HTTPSDialerPolicy = &beaconsPolicy{}

// LookupTactics implements HTTPSDialerPolicy.
func (p *beaconsPolicy) LookupTactics(ctx context.Context, domain, port string) <-chan *HTTPSDialerTactic {
	out := make(chan *HTTPSDialerTactic)

	go func() {
		defer close(out)
		index := 0

		// emit beacons related tactics first
		for tx := range p.tacticsForDomain(domain, port) {
			tx.InitialDelay = happyEyeballsDelay(index)
			index += 1
			out <- tx
		}

		// now emit tactics using the DNS
		for tx := range p.Fallback.LookupTactics(ctx, domain, port) {
			tx.InitialDelay = happyEyeballsDelay(index)
			index += 1
			out <- tx
		}
	}()

	return out
}

func (p *beaconsPolicy) tacticsForDomain(domain, port string) <-chan *HTTPSDialerTactic {
	out := make(chan *HTTPSDialerTactic)

	go func() {
		defer close(out)

		// we currently only have beacons for api.ooni.io
		if domain != "api.ooni.io" {
			return
		}

		snis := p.beaconsDomains()
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		r.Shuffle(len(snis), func(i, j int) {
			snis[i], snis[j] = snis[j], snis[i]
		})

		ipAddrs := p.beaconsAddrs()

		for _, ipAddr := range ipAddrs {
			for _, sni := range snis {
				out <- &HTTPSDialerTactic{
					Address:        ipAddr,
					InitialDelay:   0,
					Port:           port,
					SNI:            sni,
					VerifyHostname: domain,
				}
			}
		}
	}()

	return out
}

func (p *beaconsPolicy) beaconsAddrs() (out []string) {
	return append(
		out,
		"162.55.247.208",
	)
}

func (p *beaconsPolicy) beaconsDomains() (out []string) {
	// See https://gitlab.torproject.org/tpo/anti-censorship/pluggable-transports/snowflake/-/issues/40273
	return append(
		out,
		"adtm.spreadshirts.net",
		"alb.reddit.com",
		"a.loveholidays.com",
		"api.giphy.com",
		"api.nextgen.guardianapps.co.uk",
		"api.trademe.co.nz",
		"app.launchdarkly.com",
		"apps.voxmedia.com",
		"assets0.uswitch.com",
		"assets.boots.com",
		"assets.dunelm.com",
		"assets.guim.co.uk",
		"assets.hearstapps.com",
		"assets-jpcust.jwpsrv.com",
		"assets.nymag.com",
		"assets.thecut.com",
		"atreseries.atresmedia.com",
		"cdn.bfldr.com",
		"cdn.concert.io",
		"cdn.contentful.com",
		"cdn.ketchjs.com",
		"cdn.laredoute.com",
		"cdn.polyfill.io",
		"cdn.speedcurve.com",
		"cdn.sstatic.net",
		"cdn.taboola.com",
		"client.grubstreet.com",
		"client.nymag.com",
		"client-registry.mutinycdn.com",
		"client.thecut.com",
		"client.thestrategist.co.uk",
		"client.vulture.com",
		"compote.slate.com",
		"concertads-configs.vox-cdn.com",
		"contributions.guardianapis.com",
		"display.bidder.taboola.com",
		"edgemesh.webflow.io",
		"embed.api.video",
		"epsf.ticketmaster.com",
		"fastly.com",
		"fastly.jsdelivr.net",
		"fast.ssqt.io",
		"fast.wistia.com",
		"fdyn.pubwise.io",
		"fonts.nymag.com",
		"foursquare.com",
		"frend-assets.freetls.fastly.net",
		"f.vimeocdn.com",
		"github.githubassets.com",
		"global.ketchcdn.com",
		"helpersng.taboola.com",
		"hips.hearstapps.com",
		"i.guimcode.co.uk",
		"i.guim.co.uk",
		"i.insider.com",
		"images.mutinycdn.com",
		"images.taboola.com",
		"interactive.guim.co.uk",
		"i.vimeocdn.com",
		"js-agent.newrelic.com",
		"js.sentry-cdn.com",
		"linktr.ee",
		"login.nine.com.au",
		"lux.speedcurve.com",
		"martech.condenastdigital.com",
		"media0.giphy.com",
		"media1.giphy.com",
		"media2.giphy.com",
		"media3.giphy.com",
		"media.giphy.com",
		"media.newyorker.com",
		"media.wired.com",
		"mparticle.weather.com",
		"mv.outbrain.com",
		"newrelic.com",
		"next.ticketmaster.com",
		"nm.realtyninja.com",
		"pingback.giphy.com",
		"pips.taboola.com",
		"pitchfork.com",
		"pixel.condenastdigital.com",
		"player.ex.co",
		"pm-widget.taboola.com",
		"polyfill.io",
		"prd.jwpltx.com",
		"pyxis.nymag.com",
		"rapid-cdn.yottaa.com",
		"rtd-tm.everesttech.net",
		"s1.ticketm.net",
		"s3-media0.fl.yelpcdn.com",
		"slate.com",
		"sourcepoint.theguardian.com",
		"ssl.p.jwpcdn.com",
		"sstc.dunelm.com",
		"static.ads-twitter.com",
		"static.filestackapi.com",
		"static.klaviyo.com",
		"static.theguardian.com",
		"static-tracking.klaviyo.com",
		"s.w-x.co",
		"trademe.tmcdn.co.nz",
		"trc.taboola.com",
		"t.seenthis.se",
		"uploads.guim.co.uk",
		"video.seenthis.se",
		"vidstat.taboola.com",
		"vod.api.video",
		"vulcan.condenastdigital.com",
		"widget.perfectmarket.com",
		"www.allure.com",
		"www.amazeelabs.com",
		"www.architecturaldigest.com",
		"www.blackpepper.co.nz",
		"www.bonappetit.com",
		"www.cntraveler.com",
		"www.drupal.org",
		"www.dunelm.com",
		"www.epicurious.com",
		"www.fastly.com",
		"www.filestack.com",
		"www.giphy.com",
		"www.glamour.com",
		"www.gq.com",
		"www.insider.com",
		"www.jimdo.com",
		"www.loveholidays.com",
		"www.madeiramadeira.com.br",
		"www.newrelic.com",
		"www.newyorker.com",
		"www.pronovias.com",
		"www.redditstatic.com",
		"www.rvu.co.uk",
		"www.self.com",
		"www.shazam.com",
		"www.shondaland.com",
		"www.split.io",
		"www.spreadgroup.com",
		"www.spreadshirt.com",
		"www.taboola.com",
		"www.teenvogue.com",
		"www.thecut.com",
		"www.theguardian.com",
		"www.them.us",
		"www.ticketmaster.com",
		"www.trademe.co.nz",
		"www.vanityfair.com",
		"www.vogue.com",
		"www.wikihow.com",
		"www.wired.com",
		"www.yelp.com",
		"x.giphy.com",
		"yelp.com",
	)
}
