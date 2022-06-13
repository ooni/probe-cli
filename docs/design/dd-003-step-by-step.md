# Step-by-step measurements

| | |
|:-------------|:-------------|
| Author | [@bassosimone](https://github.com/bassosimone) |
| Last-Updated | 2022-06-13   |
| Reviewed-by | [@hellais](https://github.com/hellais) |
| Reviewed-by | [@DecFox](https://github.com/DecFox/) |
| Status       | approved     |
| Obsoletes | [dd-002-netx.md](dd-002-netx.md) |

*Abstract* The original [netx design document](dd-002-netx.md) is now two
years old. Since we wrote such a document, we amended the overall design
several times. The four major design changes where:

1. saving rather than emitting
[ooni/probe-engine#359](https://github.com/ooni/probe-engine/issues/359)

2. switching to save measurements using the decorator
pattern [ooni/probe-engine#522](https://github.com/ooni/probe-engine/pull/522);

3. the netx "pivot" [ooni/probe-cli#396](https://github.com/ooni/probe-cli/pull/396);

4. measurex [ooni/probe-cli#528](https://github.com/ooni/probe-cli/pull/528).

In this (long) design document, we will revisit the original problem proposed by
[df-002-netx.md], in light of what we changed and of what we learned from the
changes we applied. We will highlight the major pain points of the current
implementation, which are these the following:

1. that the measurement library API is significantly different to the Go stdlib
API, therefore violating the original `netx` design goal that writing a new
experiment means using slightly different constructors that deviate from the
standard library only to meet specific measurement goals we have;

2. that the decorator pattern leads to complexity in creating measurement types,
which in turn seems to be the cause of the previous issue;

3. that the decorator pattern does not allow us to precisely collect all the
data that matters for events such as TCP connect and DNS round trips using
a custom transport, thus suggesting that we should revisit our choice of using
decorators and revert back to some form of _constructor based injection_ to
inject a data type suitable for saving events.

In doing that, we will also propose an incremental plan for moving the tree
forward from [the current state](https://github.com/ooni/probe-cli/tree/1685ef75b5a6a0025a1fd671625b27ee989ef111)
to a state where complexity is moved from the measurement-support library to
the implementation of each individual network experiment.
