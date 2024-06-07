# Richer Input

|              |                                                |
|--------------|------------------------------------------------|
| Author       | [@bassosimone](https://github.com/bassosimone) |
| Last-Updated | 2024-06-07                                     |
| Reviewed-by  | N/A                                            |
| Status       | living document                                |

This document is a living document documenting our in-progress design
for implementing [richer input](https://github.com/ooni/ooni.org/issues/1291).

We define as richer input the possibility of providing to OONI experiments a
tuple consisting of an input string and options.

## Problem Statement

Traditionally, OONI experiments measure _inputs_. For example, the command in
Listing 1 measures the `https://www.example.com/` URL using the
`web_connectivity` experiment.

```
./miniooni web_connectivity -i https://www.example.com/
```

**Listing 1** Running Web Connectivity with a given URL using `miniooni`.

Some experiments support providing _options_ via command line. For example,
the command in Listing 2 runs the `dnscheck` experiment measuring
`https://8.8.8.8/dns-query` and using the `HTTP3Enabled` option set to `true`.

```
./miniooni dnscheck -i https://8.8.8.8/dns-query -O HTTP3Enabled=true
```

**Listing 2** Running DNSCheck with a given URL and options using `miniooni`.


Additionally, OONI Run v2 allows to run experiments with options. For example,
the JSON document in Listing 3 is equivalent to the code in Listing 2.

```JSON
{
	"nettests": [{
		"test_name": "dnscheck",
		"options": {
			"HTTP3Enabled": true
		},
		"inputs": [
			"https://8.8.8.8/dns-query"
		]
	}]
}
```

**Listing 3** Running DNSCheck with a given URL and options using an OONI Run v2 descriptor.

However, when OONI Probe runs without options, as shown in Listing 4 and
Listing 5, it either uses hardcoded defaults or OONI backend APIs to retrieve
the list of *inputs* to measure. Crucially, this list of inputs comes with
no additional options.

```
./miniooni dnscheck
```

**Listing 4** Running DNSCheck without URLs and inputs using `miniooni`.

```
ooniprobe run experimental
```

**Listing 5** Running DNSCheck indirectly through the `experimental`
suite using `ooniprobe`.

To understand what is going on, we need to briefly take a look at the types
and interfaces used by [OONI Probe v3.22.0](https://github.com/ooni/probe-cli/tree/v3.22.0).




Yet, the possibility of specifying options through the OONI backend is important to
widen the kind of network experiments we can perform and can inform decisions such as
customising the measurement algorithm and detect throttling cases.

Therefore, there is a need to remove the codebase bottlenecks preventing OONI Probe
from measuring tuples consisting of *inputs* and *options*.

We define *richer input* the combination of specific *inputs* and *options* and we say
that removing these bottlenecks *is* making richer input possible.

The rest of this document explains how we're going to do this.

