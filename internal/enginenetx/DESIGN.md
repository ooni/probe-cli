# Engine Network Extensions

This file documents the [./internal/enginenetx](.) package design. The content is current
as of [probe-cli#1552](https://github.com/ooni/probe-cli/pull/1552).

## Design Goals

We define "bridge" an IP address with the following properties:

1. the IP address is not expected to change;

2. the IP address listens on port 443 and accepts _any_ incoming SNI;

3. the webserver on port 443 proxies to the OONI APIs.

We also assume that the Web Connectivity test helpers (TH) could accept any SNIs.

Considering all of this, this package aims to:

1. overcome DNS-based censorship for "api.ooni.io" by hardcoding known-good
bridges IP addresses inside the codebase;

2. overcome SNI-based censorship for "api.ooni.io" and test helpers by choosing
from a pre-defined list of SNIs to use _instead_;

3. introduce state by remembering which tactics for creating TLS connections
have worked in the past and trying to reuse them;

4. allow for relatively fast recovery in case of network-condition changes
by remixing known-good solutions and bridge strategies with more conventional
approaches relying on using the DNS and sending the true SNI;

5. adopt a censored-users-first approach where the strategy we use by default
should allow for smooth operations _for them_ rather than prioritizing the
non-censored case and using additional tactics as the fallback;

6. try to defer sending the true `SNI` on the wire, therefore trying to
avoid triggering potential residual censorship;

7. provide a configuration file (`$OONI_HOME/engine/bridges.conf`) such that
users can manually configure TLS dialing for any backend service and third party
service that may be required by OONI Probe, therefore allowing to bypass also
IP-based restrictions as long as a known-good bridge exists.

The rest of this document explains how we designed for achieving these goals.

## High-Level API

The purpose of the `enginenetx` package is to provide a `*Network` object from which consumers
can obtain a `model.HTTPTransport` and `*http.Client` for HTTP operations:

```Go
func (n *Network) HTTPTransport() model.HTTPTransport
func (n *Network) NewHTTPClient() *http.Client
```

The returned `*http.Client` uses an internal transport, which is returned when the
package user invokes the `HTTPTransport` method. In turn, the internal transport customizes
creating TLS connections, to meet the objectives explained before.

## Creating TLS Connections

In [network.go](network.go), `newHTTPSDialerPolicy` configures the dialing policy
depending on the arguments passed `NewNetwork`:

1. if the `proxyURL` argument is not `nil`, we use the `dnsPolicy` alone;

2. othwerwise, we compose policies as illustrated by the following diagram:

```
+------------+     +-------------+     +--------------+     +-----------+
| userPolicy | --> | statsPolicy | --> | bridgePolicy | --> | dnsPolicy |
+------------+     +-------------+     +--------------+     +-----------+
```

As a first approximation, we can consider each arrow in the diagram to mean that
a policy tries to produce instructions for creating a connection and then falls back
to the subsequent policy in case it cannot generate instructions. In reality, some
policies implement a more complex strategy where they remix tactics they know and
tactics provided by the fallback, to more quickly recover from changes.

## Instructions For Dialing

Each policy implements the following interface (defined in [httpsdialer.go](httpsdialer.go)):

```Go
type httpsDialerPolicy interface {
	LookupTactics(ctx context.Context, domain, port string) <-chan *httpsDialerTactic
}
```

The `LookupTactics` operation is _conceptually_ similar to
[net.Resolver.LookupHost](https://pkg.go.dev/net#Resolver.LookupHost), because
both operations map a domain name to IP addresses to connect to. However,
there are also some key differences, namely:

1. `LookupTactics` is domain _and_ port specific, while `LookupHost`
only takes in input the domain name to resolve;

2. `LookupTactics` returns _a stream_ of viable "tactics" to establish
TLS connections, while `LookupHost` returns a list of IP addresses.

The second point, in particular, is crucial. The design of `LookupTactics` is
such that we can start attempting to dial as soon as we have some tactics
to try, while more advanced tactics are generated. A composed `httpsDialerPolicy` can,
in fact, start multiple child `LookupTactics` operations and then return them to the
caller as soon as they are ready, thus avoiding to block dialing until all of the
child operations are ready.

Also, as you may have guessed, the `dnsPolicy` is a policy that, under the hood,
eventually calls [net.Resolver.LookupHost](https://pkg.go.dev/net#Resolver.LookupHost)
to get IP addresses using the DNS used by the `engine.Session`. Typically, such a
resolver, in turn, composes several DNS-over-HTTPS resolvers with the `getaddrinfo` resolver.

A "tactic" looks like this:

```Go
type httpsDialerTactic struct {
	Address string

	InitialDelay time.Duration

	Port string

	SNI string

	VerifyHostname string
}
```

Here's an explanation of why we have each field in the struct:

- `Address` and `Port` qualify the TCP endpoint;

- `InitialDelay` allows a policy to delay a connect operation to implement
something similar to [happy eyeballs](https://en.wikipedia.org/wiki/Happy_Eyeballs);

- `SNI` is the `SNI` to send as part of the TLS ClientHello;

- `VerifyHostname` is the hostname to use for TLS certificate verification.

The separation of `SNI` and `VerifyHostname` is what allows us to send an innocuous
SNI over the network and then verify the certificate using the real SNI.

## HTTPS Dialer

Creating TLS connections is implemented by `(*httpsDialer).DialTLSContext`, also
part of [httpsdialer.go](httpsdialer.go). This method _morally_ implements the following
algorithm (where we omitted error handling and returning a conn for simplicity):

```Go
index := 0
for tx := range policy.LookupTactics() {
	// avoid trying the same policy twice
	if isDuplicate(tx) {
		continue
	}

	// create delay for this tactic
	delay := happyEyeballsDelay(index)
	index++

	// dial in a background gorountine (simplified algorithm)
	go func(tx, delay) {
		time.Sleep(delay)
		conn := tcpConnect(tx.Address, tx.Port)
		tconn := tlsHandshake(conn, tx.SNI, false /* skip verification */)
		verifyHostname(tlsConn, tx.VerifyHostname)
	}(tx, delay)
}
```

When a connection attempt succeds, we use cancellable `context.Context` to cancel
all the other connect attempts that may be in progress (not shown in the above
algorithm for simplicity). If all connection attempts fail, instead, we return a
composed error (again, not showed above for simplicity).

By using a modified happy eyeballs with baseline values that take into account
the overall time to perform a TLS handshake, we attempt to strike a balance
between simplicity (i.e., running operations sequentially), performance (running
them in parallel) and network load (hence the usage of happy eyeballs).

Additionally, the dialing algorithm keeps statistics about the operations it
performs using an `httpsDialerEventsHandler` type:

```Go
type httpsDialerEventsHandler interface {
	OnStarting(tactic *httpsDialerTactic)
	OnTCPConnectError(ctx context.Context, tactic *httpsDialerTactic, err error)
	OnTCPConnectSuccess(tactic *httpsDialerTactic)
	OnTLSHandshakeError(ctx context.Context, tactic *httpsDialerTactic, err error)
	OnTLSVerifyError(tactic *httpsDialerTactic, err error)
	OnSuccess(tactic *httpsDialerTactic)
}
```

These statistics contribute to construct knowledge about the network
conditions and influence the choice of policies.

## dnsPolicy

The `dnsPolicy` is implemented by [dnspolicy.go](dnspolicy.go).

Its `LookupTactics` algorithm is quite simple:

1. we arrange for short circuiting cases in which the `domain` argument
contains an IP address to "resolve" exactly that IP address (thus emulating
what `getaddrinfo` would do and avoiding to call onto the more-complex
underlying composed DNS resolver);

2. for each resolved address, we generate tactics in the most straightforward
way, e.g., where the `SNI` and `VerifyHostname` equal the `domain`.

Using this policy alone is functionally equivalent to combining a DNS lookup
operation with TCP connect and TLS handshake operations.

## userPolicy

The `userPolicy` is implemented by [userpolicy.go](userpolicy.go).

When constructing a `userPolicy` with `newUserPolicy` we indicate a fallback
`httpsDialerPolicy` to use if there is no `$OONI_HOME/engine/bridges.conf` file.

As of 2024-04-16, the structure of such a file is like in the following example:

```JSON
{
	"DomainEndpoints": {
		"api.ooni.io:443": [{
			"Address": "162.55.247.208",
			"Port": "443",
			"SNI": "www.example.com",
			"VerifyHostname": "api.ooni.io"
		}]
	},
	"Version": 3
}
```

The `newUserPolicy` constructor reads this file from disk on startup
and keeps its content in memory.

`LookupTactics` will:

1. check whether there's an entry for the given `domain` and `port`
inside the `DomainEndpoints` map;

2. if there are no entries, fallback to the fallback `httpsDialerPolicy`;

3. otherwise return all the tactic entries.

Because `userPolicy` is user-configured, we _entirely bypass_ the
fallback policy when there's an user-configured entry.

## statsPolicy

The `statsPolicy` is implemented by [statspolicy.go](statspolicy.go).

The general idea of this policy is that it depends on:

1. a `*statsManager` that keeps persistent stats about tactics;

2. a "fallback" policy.

In principle, one would expect `LookupTactics` to first return all
the tactics we can see from the stats and then try tactics obtained
from the fallback policy. However, this simplified algorithm would
lead to suboptimal results in the following case:

1. say there are 10 tactics for "api.ooni.io:443" that are bound
to a specific bridge address that has been discontinued;

2. if we try all these 10 tactics before trying fallback tactics, we
would waste lots of time failing before falling back.

Conversely, a better strategy is to remix tactics as implemented
by the [remix](remix.go) file:

1. we take the first two tactics from the stats;

2. then we take the first two tactics from the fallback;

3. then we remix the rest, not caring much about whether we're
reading from the stats of from the fallback.

Because we sort tactics from the stats by our understanding of whether
they are working as intended, we'll prioritize what we know to be working,
but then we'll also throw some new tactics into the mix.

As an additional optimization, when reading from the fallback, the
`statsPolicy` will automatically exclude TCP endpoints that have
failed recently during their TCP connect stage. If an IP address seems
IP blocked, it does not make sense to continue wasting time trying
to connect to it (a timeout is in the order of ~10s).

## bridgePolicy

The `bridgePolicy` is implemented by [bridgespolicy.go](bridgespolicy.go) and
rests on the assumptions made explicit in the design section. That is:

1. that there is a _bridge_ for "api.ooni.io";

2. that the Web Connectivity Test Helpers accepts any SNI.

Here we're also using the [remix.go](remix.go) algorithm to remix
two different sources of tactics:

1. the `bridgesTacticsForDomain` only returns tactics for "api.ooni.io"
using existing knowledge of bridges and random SNIs;

2. the `maybeRewriteTestHelpersTactics` method filters the results
coming from the fallback tactic such that, if we are connecting
to a known test-helper domain name, we're trying to hide its SNI.

## Overall Algorithm

**TODO(bassosimone)**: adapt the mixing algorithm to do exactly
this and make sure there are tests for this.

Having discussed all the polices in isolation, it now seems useful
to describe what is the overall algorithm we want to achieve:

1. if there is a `$OONI_HOME/engine/bridges.conf` with a valid entry
for the domain and port, use it without trying subsequent tactics;

2. use the first two tactics coming from stats, if any;

3. then use the first two tactics coming from bridges, if any;

4. then use the first two tactics coming from the DNS;

5. after that, randomly remix the remaining tactics.

Now, it only remains to discuss managing stats.

## Managing Stats

TODO
