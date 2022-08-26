# webconnectivity

This directory contains a new implementation of [Web Connectivity](
https://github.com/ooni/spec/blob/master/nettests/ts-017-web-connectivity.md).

As of 2022-08-26, this code is experimental and is not selected
by default when you run the `websites` group. You can select this
implementation with `miniooni` using `miniooni web_connectivity@v0.5`
from the command line.

Issue [#2237](https://github.com/ooni/probe/issues/2237) explains the rationale
behind writing this new implementation.

## Implementation overview

The experiment measures a single URL at a time. The OONI Engine invokes the
`Measurer.Run` method inside the [measurer.go](measurer.go) file.

This code starts a number of background tasks, waits for them to complete, and
finally calls `TestKeys.finalize` to finalize the contet of the JSON measurement.

The first task that is started is deals with DNS and lives in the
[dnsresolvers.go](dnsresolvers.go) file. This task is responsible for
resolving the domain inside the URL into `0..N` IP addresses.

The domain resolution includes the system resolver and a DNS-over-UDP
resolver. The implementaion _may_ do more than that, but this is the
bare minimum we're feeling like documenting right now. (We need to
experiment a bit more to understand what else we can do there, hence
the code is _probably_ doing more than just that.)

Once we know the `0..N` IP addresses for the domain we do the following:

1. start a background task to communicate with the Web Connectivity
test helper, using code inside [control.go](control.go);

2. start an endpoint measurement task for each IP adddress (which of
course only happens when we know _at least_ one addr).

Regarding starting endpoint measurements, we follow this policy:

1. if the original URL is `http://...` then we start a cleartext task
and an encrypted task for each address using ports `80` and `443`
respectively.

2. if it's `https://...`, then we only start encrypted tasks.

Cleartext tasks are implemented by [cleartextflow.go](cleartextflow.go) while
encrypted tasks live in [secureflow.go](secureflow.go).

A cleartext task does the following:

1. TCP connect;

2. additionally, the first task to establish a connection also performs
a GET request to fetch a webpage.

An encrypted task does the following:

1. TCP connect;

2. TLS handshake;

3. additionally, the first task to handshake also performs
a GET request to fetch a webpage _iff_ the input URL was `https://...`

If fetching the webpage returns a redirection, we start a new DNS task passing it
the redirect URL as the new URL to measure. We do not call the test helper again
when this happens, though. The Web Connectivity test helper already follows the whole
redirect chain, so we would need to change the test helper to get information on
each flow. When this will happen, this experiment will probably not be Web Connectivity
anymore, but rather some form of [websteps](https://github.com/bassosimone/websteps-illustrated/).

Additionally, when the test helper terminates, we run TCP connect and TLS handshake
(when applicable) for new IP addresses discovered usiong the test helper that were
previously unknown to the probe, thus collecting extra information.

As previously mentioned, when all tasks complete, we call `TestKeys.finalize`.

In turn, this function analyzes the collected data:

- [analysiscore.go](analysiscore.go) contains the core analysis algorithm;

- [analysisdns.go](analysisdns.go) contains DNS specific analysis;

- [analysishttpcore.go](analysishttpcore.go) contains the bulk of the HTTP
analysis, where we mainly determine TLS blocking;

- [analysishttpdiff.go](analysishttpdiff.go) contains the HTTP diff algorithm;

- [analysistcpip.go](analysistcpip.go) checks for TCP/IP blocking.

## Data model

All the test keys that are not part of the original Web Connectivity have
the `x_` prefix to indicate that they are experimental and may change.

## Limitations and next steps

We need to extent the Web Connectivity test helper to return us information
about TLS handshakes with IP addresses discovered by the probe. This information
would allow us to make more precise TLS blocking statements.

Further changes are probably possible. Departing too radically from the Web
Connectivity model, will lead us to have a Websteps implementation (but then
the data model would probably be different).
