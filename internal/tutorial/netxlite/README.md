# Tutorial: using the netxlite networking library

Netxlite is the underlying networking library we use in OONI. In
most cases, network experiments do not use netxlite directly, rather
they use abstractions built on top of netxlite. Though, you need to
know about netxlite if you need to modify these abstractions.

For this reason, this chapter shows the basic netxlite primitives
that we use when writing higher-level measurement primitives.

