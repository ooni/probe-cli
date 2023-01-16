# Tutorials: writing OONI nettests

This package contains a living tutorial explaining how to write OONI
nettests. The code in here is based on existing nettests.

Because it's committed to the probe-cli repository and depends on
real OONI code, it should always be up to date.

## Index

- [Rewriting the torsf experiment](experiment/torsf/): this tutorial
explains to you how to write a simple experiment. After reading it, you
will understand the interfaces between an experiment and the OONI
core. What this tutorial does not teach you, though, is how
to tell the OONI core about this experiment. To see how to do that,
you should check how we do that in [internal/registry](../registry).

- [Using the measurex package to write network experiments](measurex): this
tutorial explains to you how to use the `measurex` library to write networking
code that generates measurements using the OONI data format. You will learn
how to perform DNS, TCP, TLS, QUIC, HTTP, HTTPS, and HTTP3 measurements.

- [Low-level networking using netxlite](netxlite): this tutorial introduces
you to the `netxlite` networking library. This is the underlying library
used by `measurex` as well as by many other libraries inside OONI. You need
to know about this library to contribute to `measurex` as well as to other
parts of the OONI core that perform network operations.

Therefore, after reading these tutorials, you should have a better
understanding of how an experiment interacts with the OONI core, as
well as of which libraries you can use to write experiments.

## Regenerating the tutorials

```
(cd ./internal/tutorial && go run ./generator)
```
