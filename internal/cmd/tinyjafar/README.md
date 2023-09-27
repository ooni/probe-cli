# internal/cmd/tinyjafar

This directory builds a program you can use to provoke simple network
interference conditions, as described in detail below.

To build, use:

```console
go build -v ./internal/cmd/tinyjafar
```

Any requirement that applies to building OONI Probe also applies to
building this small helper program, since they use the same base library.

The command line interface is backwards compatible with the one
implemented by [the original jafar](https://github.com/ooni/probe-cli/tree/v3.18.1/internal/cmd/jafar)
except that `tinyjafar` only supports iptables flags.

To use this tool, you must be on Linux and have iptables installed. We do not
use this tool for QA, but it is mentioned in [tutorials](../../../internal/tutorial/).

## Drop traffic towards a given IP address

In one console, run:

```console
./tinyjafar -iptables-drop-ip 130.192.16.171
```

The program will run some `iptables` commands showing each command
it runs. These commands configure `iptables` to block some internet traffic
and the blocking will stay in place until you interrupt
`tinyjafar` using Ctrl-C. When existing, `tinyjafar` will
undo all the commands it executed when starting up.

While `tinyjafar` is running, in another console try this command:

```console
curl -v https://nexa.polito.it/
```

If the IP address has not changed since writing this README, the
`curl` command should eventually timeout when connecting.

## Drop packets containing an hex sequence

```console
./tinyjafar -iptables-drop-keyword-hex "|07 65 78 61 6d 70 6c 65 03 63 6f 6d|"
```

and

```console
dig @8.8.8.8 www.example.com
```

The `tinyjafar` invocation drops DNS queries for `www.example.com`.

## Drop packets containing a string

```console
./tinyjafar -iptables-drop-keyword ooni.org
```

and

```console
curl -v https://ooni.org/
```

We expect cURL to timeout during the TLS handshake since we're
blocking the string that appears in the SNI field.

## Preventing TCP-connecting to a host

```console
./tinyjafar -iptables-reset-ip 130.192.16.171
```

and

```console
curl -v https://nexa.polito.it/
```

This should fail with "connection refused".

## Resetting a TCP connection containing an hex pattern

```console
./tinyjafar -iptables-reset-keyword-hex "|6F 6F 6E 69|"
```

and

```console
curl -v https://ooni.org/
```

This should reset the TCP connection because the TLS Client Hello
contains "ooni" (`6F 6F 6E 69` in hex).

## Resetting a TCP connection containing a string pattern

`console
./tinyjafar -iptables-reset-keyword ooni
```

and

```console
curl -v https://ooni.org/
```
