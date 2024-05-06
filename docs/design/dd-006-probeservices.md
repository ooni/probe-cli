# Probe Services Interactions

|              |                                                |
|:-------------|:-----------------------------------------------|
| Author       | [@bassosimone](https://github.com/bassosimone) |
| Last-Updated | 2024-05-06                                     |
| Status       | documentational                                |

*Abstract.* We document the interaction with the probe services.

## Introduction

While running OONI Probe needs to communicate with several services, some
of which are managed by OONI, some of which by third parties. This document is
about explaining the interaction with internal services.

Services managed by OONI are divided into two distinct categories:

1. "probe services", that is APIs that provide services to OONI probe
during its bootstrap, when fetching inputs, or when submitting measurements;

2. "test helpers", that is APIs invoked while measuring.

This document is mostly concerned with the "probe services". Historically,
the probe services were implemented by separate hosts. However, during the
Summer 2019 team meeting in Stockholm, we decided to consolidate all of
them and serve them through a single entry point. Most recently, this entry
point has been implemented by the `api.ooni.io` host.

## Software Architecture

There are OONI Probe Mobile, OONI Probe Desktop, and OONI Probe CLI. Also, there
are two CLI clients, `ooniprobe` and `miniooni` (the research client).

Mobile clients used the [pkg/oonimkall](../../pkg/oonimkall/) API. Desktop
clients invoke `ooniprobe`. Both `ooniprobe` and `miniooni` directly use the
[internal/engine](../../internal/engine/) API. Additionally, `minniooni`
implements the OONI Run v2 preview using [internal/oonirun](../../internal/oonirun).

The following diagram summarizes what we said so far:

```
  +-----------------+ +---------------+ +----------------------------+
  |  Probe Mobile   | | Probe Desktop | |          miniooni          |
  +-----------------+ +---------------+ +----------------------------+
           |                  |               |
		   V                  V               |
  +-----------------+ +---------------+       |     +----------------+
  |  oonimkall API  | |   ooniprobe   |       |     |   oonirun API  |
  +-----------------+ +---------------+       |     +----------------+
           |                  |               |             |
		   V                  V               V             V
  +------------------------------------------------------------------+
  |                             Engine API                           |
  +------------------------------------------------------------------+
```

In other words, the engine API mediates all interactions. (This is true not
only for communicating with the probe services, but in general.)

## The Engine API

The `*engine.Session` type represents a measurement session. This type
provides APIs to higher-level blocks in the software architecture, as
described above. In addition, it provides individual network experiments
with services and state. For example, it constructs an HTTP client with
possibly circumvention features, to communicate with test helpers. The
session also implements all the functionality to communicate with the probe
services (e.g., the functionality to invoke the check-in API).

From the engine's point of view, the software architecture is the following:

```
  +------------------------------------------------------------------+
  |                       .../engine (Engine API)                    |
  +------------------------------------------------------------------+
           |                     |                      |
		   V                     V                      V
  +-----------------+ +---------------------+ +----------------------+
  | .../enginenetx  | |  .../engineresolver | | .../probeservices    |
  +-----------------+ +---------------------+ +----------------------+
                                                        |
														V
                                              +----------------------+
                                              |   .../httpclientx    |
										      +----------------------+
```

The `.../` ellipsis indicates packages inside [internal](../../internal/).

Basically:

1. the `engineresolver` package manages a composed DNS-over-HTTPS
resolver that falls back to the system resolver;

2. the `enginenetx` package manages dialing TLS connections
and is where we implement the "bridges" circumvention strategy (see
the package's design document for more information);

3. the `engine` API uses `engineresolver` and `enginenetx`
to create an HTTP client with extra robustness properties compared to
the one provided by the Go standard library;

4. the `probeservices` package is where we use such an HTTP
client to communicate with the probe services;

5. in turn `probeservices` uses the `httpclientx` package implements
algorithms for communicating with the probe services (and other services),
including among them the possibility of trying a set of equivalent URLs
in ~parallel (refer to the package's design document for more information).

In other words:

1. `engineresolver` and `enginenetx` provide an HTTP client, that
is instantiated by the engine API;

2. `httpclientx` provides algorithms to use such a client;

3. `probeservices` uses `httpclientx` algorithms and the given client
to implement communication with the probe services, but all the
OONI Probe's code uses the `probeservices` package through the engine API.

## Conclusion

Probe services are a set of APIs used by OONI Probe through its
lifecycle. Every client communicates with these services through
the "engine" API. In turn, the "engine" API uses the support
"probeservices" package to communicate with the probe services. In
turn, "probeservices" uses algorithms defined by "httpclientx" as
well as an HTTP client created by the "engine" API using the
"engineresolver" and "enginenetx" packages.
