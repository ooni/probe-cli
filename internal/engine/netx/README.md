# Package github.com/ooni/probe-engine/netx

OONI extensions to the `net` and `net/http` packages. This code is
used by `ooni/probe-engine` as a low level library to collect
network, DNS, and HTTP events occurring during OONI measurements.

This library contains replacements for commonly used standard library
interfaces that facilitate seamless network measurements. By using
such replacements, as opposed to standard library interfaces, we can:

* save the timing of HTTP events (e.g. received response headers)
* save the timing and result of every Connect, Read, Write, Close operation
* save the timing and result of the TLS handshake (including certificates)

By default, this library uses the system resolver. In addition, it
is possible to configure alternative DNS transports and remote
servers. We support DNS over UDP, DNS over TCP, DNS over TLS (DoT),
and DNS over HTTPS (DoH). When using an alternative transport, we
are also able to intercept and save DNS messages, as well as any
other interaction with the remote server (e.g., the result of the
TLS handshake for DoT and DoH).

This package is a fork of [github.com/ooni/netx](https://github.com/ooni/netx).
