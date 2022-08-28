# Tutorial: using the netxlite networking library

Netxlite is the underlying networking library we use in OONI. In
most cases, network experiments do not use netxlite directly, rather
they use abstractions built on top of netxlite (e.g., measurex).
Though, you need to know about netxlite if you need to modify
these abstractions.

For this reason, this chapter shows the basic netxlite primitives
that we use when writing higher-level measurement primitives.

We will start from simple primitives and we will combine them
together to reach to the point where we can perform GET requests
to websites using already existing TLS or QUIC connections. (The code
we will end up writing will look like a stripped down version of
the measurex library, for which there is a separate tutorial.)

Index:

- [chapter01](chapter01) shows how to establish TCP connections;

- [chapter02](chapter02) covers TLS handshakes;

- [chapter03](chapter03) discusses TLS parroting;

- [chapter04](chapter04) shows how to establish QUIC sessions;

- [chapter05](chapter05) is about the "stdlib" DNS resolver;

- [chapter06](chapter06) discusses custom DNS-over-UDP resolvers;

- [chapter07](chapter07) shows how to perform an HTTP GET
using an already existing TLS connection to a website;

- [chapter08](chapter08) is like chapter07 but for QUIC.
