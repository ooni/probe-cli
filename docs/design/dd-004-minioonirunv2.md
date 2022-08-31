# Mini OONIRun v2

|              |                                                |
|--------------|------------------------------------------------|
| Author       | [@bassosimone](https://github.com/bassosimone) |
| Last-Updated | 2022-08-31                                     |
| Reviewed-by  | [@hellais](https://github.com/hellais)         |
| Status       | approved                                       |

The official spec for OONI Run v2 is under review as [ooni/spec#249](
https://github.com/ooni/spec/pull/249). We want to start experimenting
with OONI Run v2 before such a spec has been finalized.

This document, in particular, describes a "mini" OONI Run v2 where
`miniooni` users can run arbitrary OONI Run v2 "descriptors" (see below
for a definition).

When [ooni/spec#249](https://github.com/ooni/spec/pull/249) is merged,
it will automatically supersede this document. The main idea of this
document is just to explain the subset of OONI Run v2 implemented inside
the `miniooni` research tool _until_ we have a complete spec.

## Problem statement

OONI Run v1 links do not support passing options to experiments and
often times are longer than the maximum message size on, for example,
WhatsApp and Telegram. To fix these issues, we're designing OONI Run v2.

The general idea is the following:

* there is a mobile (or desktop) OONI Run v2 deeplink;

* the deeplink eventually resolves to a URL;

* fetching the URL returns a JSON document telling the OONI engine
what to measure, called _descriptor_.

This document only focuses on running arbitrary v2 descriptors
for research purposes  using `miniooni`.

## OONI Run v2 descriptor

We use this definition of the descriptor (which we deemed to be
forward compatible with the _final_ descriptor format):

```JSON
{
	"nettests": []
}
```

where `"nettests"` is an array of objects like this:

```JSON
{
	"inputs": [],
	"options": {},
	"test_name": ""
}
```

where:

- `"inputs"` is an optional list of input strings for the nettest like the ones you
would normally pass to `miniooni` using `-i`.

- `"options"` is an optional `map[string]any` where the key is a name of an option
you can pass to a `miniooni` experiment using `-O NAME=VALUE` and `value` is the
corresponding value. The type of the value should be the type of the original field
inside of the experiment's options, which you can see with `./miniooni <experiment> --help`.

- `"test_name"` is the `<experiment>` string you would use in `./miniooni <experiment>`.

(For historical reasons, we use the terms "experiment", "test", and "nettest"
interchangeably to refer to the network tests run by OONI.)

## Functionality

A `miniooni` user could run an arbitrary OONI Run v2 descriptor stored at a given URL
by invoking the `oonirun` subcommand and passing the URL using `-i`. The user could
specify an abitrary number of descriptor URLs. For example:

```bash
./miniooni oonirun -i https://example.com/a -i https://example.com/b
```

The OONI engine will fetch each descriptor _sequentially_ and execute the code it
would execute if the user ran `./miniooni` from the command line
with the specified experiment name, the given
options and inputs.

The OONI engine stores the content of each descriptor inside a cache on disk
inside the `$OONI_HOME` of `miniooni` (generally `$HOME/.miniooni/`).

Conceptually, this cache is a map from the descriptor URL to the latest known content
of the descriptor. (The actual implementation _may_ differ.)

On the first run, or when the descriptor has changed, `miniooni` refuses to run,
shows what changed, and asks the user for confirmation.

There's also a mechanism to bypass asking for confirmation that explicitly requires a
user to add `-y` or `--yes` to the command line to automatically answer "yes" to
all questions. (This is useful when, for example, you're running your own descriptors.)
