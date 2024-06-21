# Throttling measurement methodology

|              |                                                |
|--------------|------------------------------------------------|
| Author       | [@bassosimone](https://github.com/bassosimone) |
| Last-Updated | 2024-06-21                                     |
| Reviewed-by  | [@DecFox](https://github.com/DecFox)           |
| Status       | accepted                                       |

This document explains the throttling measurement methodology implemented inside
the [ooni/probe-cli](https://github.com/ooni/probe-cli) repository.

We are publishing this document as part of this repository for discussion. A future
version of this document may be moved into the [ooni/spec](https://github.com/ooni/spec)
repository.

## Problem statement

We are interested to detect cases of _extreme throttling_. We say that throttling is
_extreme_ when the speed to access web resources is _significantly reduced_ (10x or more)
compared to what is _typically_ observed. We care about extreme throttling because we
are interested in cases in which the performance impact is such to make the website
_unlikely_ to work as intended for web users in a country.

Additionally, as recently discussed with [@inetintel](https://github.com/InetIntel/)
researchers et al., we are interested to detect cases of _targeted throttling_. That is
cases where a set of specific websites gets significantly worse performance while the
overall users' internet experience is unchanged. This kind of throttling is in opposition
to _generalized throttling_ where the internet experience is degraded regardless of the
website compared to the previous internet experience (see [Dimming the Internet by Collin
Anderson](https://censorbib.nymity.ch/#Anderson2013a) for seminal work on this topic).

We, and other researchers, have documented extreme, targeted throttling in the
past. See, for example:

1. [our blog post documenting twitter throttling in Russia](
https://ooni.org/post/2022-russia-blocks-amid-ru-ua-conflict/), which is the
first instance in which we tested this methodology.

2. [our blog post documenting throttling in Kazakhstan](
https://ooni.org/post/2023-throttling-kz-elections/).

3. ["Throttling Twitter: an emerging censorship technique in Russia" by Xue et al.](
https://censorbib.nymity.ch/#Xue2021a).

OONI Probe measures websites as part of the [Web Connectivity experiment](
https://github.com/ooni/spec/blob/master/nettests/ts-017-web-connectivity.md) and
these measurements contain peformance metrics.

The next section explains which performance metrics we collect and how these can
be useful to document episodes of extreme, targeted throttling.

## Methodology

The overall idea of our methodology is that, as a first approximation,
we're not concerned with _how_ throttling is implemented, rather we aim to
show clearly-degraded network performance.

We aim to detect such a degradation by comparing metrics collected by OONI Probe instances
running in a country and network with measurements previously collected by users and/or with
concurrent measurements towards different targets.

### Network Events

Web Connectivity v0.5 collects the first 64 [network events](
https://github.com/ooni/spec/blob/master/data-formats/df-008-netevents.md) occurring
on a given TCP connection. These events include "read" and "write" events, which
map directly to network I/O operations (i.e., the `recv` and `send` syscalls
respectively). We focus on throttling in the download direction, therefore we're
mostly interested into "read" events.

The basic structure of a "read" network events is the following:

```JSON
{
    "address": "1.1.1.1:443",
    "failure": null,
    "num_bytes": 4114,
    "operation": "read",
    "proto": "tcp",
    "t0": 1.001,
    "t": 1.174,
    "tags": [],
    "transaction_id": 1,
}
```

Through these events, we know when "read" returned (`t`), for how much time it was blocked
(`t - t0`), and the number of bytes received (`num_bytes`).

The slope of the integral of "read" events, provides information about the speed
at which we were receiving data from the network. Slow downs in the stream either correspond
to reordering and retransmission events (where there is head-of-line blocking) or to
timeout events (where the TCP pipe is empty).

Additionally, network events contain events such as `"tls_handshake_start"` and
`"tls_handshake_done`", which look like the following:

```JSON
{
    "address": "1.1.1.1:443",
    "failure": null,
    "num_bytes": 0,
    "operation": "tls_handshake_start",
    "proto": "tcp",
    "t0": 1.001,
    "t": 1.001,
    "tags": [],
    "transaction_id": 1,
}
```

These events allow us to know when we started and we stopped handshaking.

Now, considering that the amount of bytes transferred by a TLS handshake with the
same server using similar client code is not far from being constant (i.e., it's a relatively
narrow gaussian with small sigma), we can model the problem of TLS handshaking as
the problem of downloading a ~fixed amount of data.

With many users measuring popular websites using OONI Probe in a given country
and network, we can therefore establish comparisons of current performance metrics with
previous performance metrics. In case of extreme throttling, where the download speed
is reduced of at least 10x, we would notice a performance difference. The _time_
required to complete the TLS handshake should be a sufficient metric (and, in fact,
_is_ a performance metric used by speed tests such as
[speed.cloudflare.com](https://speed.cloudflare.com/)).

Additionally, in Web Connectivity v0.5, the "read" events data collection does not
stop after the TLS handshake, therefore, we will have several post-handshake data
points we could also use to make statements about throttling. The size of the webpage
fetched from a given country and network, in fact, should also be pretty constant,
so a reasoning similar to the one made above for the TLS handshake also applies to the
process of handshadking and then downloading a web page. However, because very long
downloads could collect lots of "read" events, and because we want to limit the maximum
amount of "read" events we collected to 64, we have also introduced the following,
complementary metric to investigate throttling.

### Download speed metrics

Web Connectivity v0.5 also collects download speed samples for connections
used to access websites. We use the same methodology used by [ndt7](
https://github.com/m-lab/ndt-server/blob/main/spec/ndt7-protocol.md). We measure
the cumulative number of bytes received by a connection using a truncated exponential
distribution to decide when to collect samples. By not collecting samples at fixed
intervals, we [should have PASTA properties](https://en.wikipedia.org/wiki/Arrival_theorem#Theorem_for_arrivals_governed_by_a_Poisson_process).

The total TLS handshaking, HTTP round trip and body fetching time is bounded by a fixed amount of
seconds (currently ten seconds for the handshake and ten additional seconds for HTTP). Additionally,
there is a cap on the maximum amount of body bytes to download (`1<<19`).

The expected size of a downloaded webpage should be pretty constant for clients
attempting to fetch such a webpage from the same country and network. Therefore, we
can model handshaking plus fetching the body as asking the question of how much
time it takes to download `handshake_size + min(body_size, 1<<19)` bytes in up to
~twenty seconds.

If we assume that the server is not going to throttle downloads (which is still
an hypothesis worth considering), save for some (healthy) packet pacing, significant
changes in the _time_ required to perform the whole set of operations would be
an indication of extreme throttling. However, in using time as the metric, or any
other metric, we need to remember to classify measurements that time out (i.e., are
not able to fetch the whole body) apart from the ones that complete successfully.

Those measurements, in fact, should not be considered "failed" for the purpose of
measuring throttling. Rather, if the TCP connection could progress into the handshake
and possibily into downloading a webpage, these measurements would possibly be
an additional indication of extreme, targeted throttling.

## Discussion

This methodology leverages existing performance metrics inside of Web Connectivity
v0.5 to passively detect extreme throttling. Because this methodology models
the TLS handshake and fetching the body as speed tests, it is, however, not possible
to provide users with clear indication of throttling after a single run. We will,
instead, need to collect several samples over time and cross compare them using
the [ooni/data](https://github.com/ooni/data) measurement pipeline.

More specifically, we would need to compare current measurements with past
measurements collected for the same target website by users living in the same
country and using the same autonomous system. Alternatively, we could compare
measurements collected during the same time frame towards different websites, even
though this signal is weaker because it can just be caused by interconnection
issues. In any case, these considerations imply that our methodology rests
on the assumption that we will have several measurements for the targer websites,
and our confidence would clearly lower with little available data.

In analysing the data, it would also be useful to consider the possibility of
checking whether specific HTTP headers or the host name (after a redirect) clearly
indicate specific geographic locations. For example, Cloudflare includes a
`cf-ray` header indicating the specific cache that is serving the content using
the name of the nearest airport.

Additionally, with the availability of
[richer input](https://github.com/ooni/probe-cli/blob/master/docs/design/dd-008-richer-input.md),
it would become possible to
run custom `urlgetter` experiments where we use possibly offending and possibly not
offending SNIs with target addresses and possibly-unrelated addresses, thus giving
us a chance to narrow down the cause of throttling to, say, the SNI being used.

Throttling could be caused by policers and shapers as well as by forcing specific
users to pass through a congested path. When policers and shapers are used, we
expect the speed to likely converge to predictable values (e.g., 128 kbit/s). On the
contrary, when throttling is driven by congestion, we expect to see higher variance
in the results, possibly correlated with daily usage patterns.

## Digital Divide Implications

By collecting passive performance metrics, we are not only equipped to detect
extreme, targeted throttling, but we are also gathering information about the performance
achievable by clients in several world regions for reaching specific websites. The
availability of HTTP headers and the practice of some CDNs of annotating the
responses with headers indicating which specific cache is being (as mentioned above
in the case of Cloudflare) used could also be exploited to make interesting
digital-divide statements.

## Future Work

With network events, we can also collect some ~baseline RTT samples. The `t - t0` time
of the TCP connect event provides an upper bound of the path RTT _unless_ there is a
retransmission of the `SYN` segment. The TLS handshake also involves sending TCP segments
back and forth in such a fashion that it's possible to extract RTT metrics. Howewer, we
should be careful to exclude segments sent back to back.

In general, detecting more precisely the characteristics of throttling either
requires additional research aimed at classifying the stream of events emitted
by a receiving socket under specific throttling conditions. A possible starting
point for this research could be ["Strengthening measurements from the edges:
application-level packet loss rate estimation" by Basso et al.](
https://www.sigcomm.org/sites/default/files/ccr/papers/2013/July/2500098-2500104.pdf).

An alternative approach, already mentioned above, would require the possibility
of providing OONI experiments such as `urlgetter` with
[richer input](https://github.com/ooni/probe-cli/blob/master/docs/design/dd-008-richer-input.md)
parameters that
could provide additional data to answer more-narrow research questions. For example, if
there are reports that a website is throttled by SNI, we could perform a download
from a given test server with certificate verification disabled, using the offending
SNI and an innocuous SNI.

Because HTTP/3 used QUIC and because QUIC operates in userspace, there is
also the possibility of instrumenting the QUIC library to periodically collect
snapshots about the receiver's state. However, in general, sender stats are
much more useful to understand QUIC performance. This fact implies that we could
instrument a QUIC library to observe the sender's state and gather information
about throttling uploads. (However, the whole design of Web Connectivity is not
such that we upload resources, therefore we would need to figure out whether
it is possible to overcome this fundamental limitation first.)

In the same vein, our Web Connectivity methodology does not currently factor in
the possibility of measuring upload speed throttling for HTTP/1.1 and HTTP/2. However,
anecdotal evidence exists that some countries may throttle the upload path or just
have poor upstream connectivity towards interesting websites. A technique that
has sometimes been applied is that of including very large headers into the request
body, even though servers may not necessarily accept such headers.
