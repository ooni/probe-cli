# Gardener design

The gardener should help researchers to curate the
[test-lists](https://github.com/citizenlab/test-lists).

The [test-lists](https://github.com/citizenlab/test-lists) are
the input to ooniprobe measurements. We run longitudinal measurements
over a long time period to keep track of how censors censor the
internet. Changing the test lists could impact onto the aggregate
confirmed, anomaly, success, failure fractions (as shown by the
[MAT](https://explorer.ooni.org/chart/mat) and the
[API](https://api.ooni.io/)).

Additionally, the Web Connectivity measurement algorithm and
the prioritization algorithm might compensate for issues inside
the test lists, so we need to ensure the policies for updating
test lists evolve along with these algorithms.

## General principles

We SHOULD take recent measurements into account before removing
entries from the [test-lists](https://github.com/citizenlab/test-lists). If
the [aggregation API](https://api.ooni.io/apidocs/#/default/get_api_v1_aggregation)
says that there is censorship for a specific URL somewhere in the world,
we MAY choose to keep the URL inside the test lists.

## Expired domains

The test lists contain several domains that no longer exist. We can
check the whole test list by using the same DNS resolver use use
in the Web Connectivity test helper, i.e., [dns.google](https://dns.google/).

The `gardner dnsreport` subcommand helps us to determine which domains
do not exist anymore. They will have a `dns_nxdomain_error` failure.

The choice on whether to keep a domain is ultimately a subjective decision
of the researcher updating the test lists. Because of the rationale set
forth in the general principles, the gardner provides information collected
using the [aggregation API](https://api.ooni.io/apidocs/#/default/get_api_v1_aggregation)
side by side with DNS results, to help making informed decisions.

The `gardener dnsfix` command helps the researcher in case there are many
URLs with expired domains by applying simple rules to remove URLs in the
most _obvious_ cases (i.e., no anomaly or confirmed for such a URL in the
last month).
