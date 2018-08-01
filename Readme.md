# OONI Probe CLI

The next generation OONI Probe Command Line Interface.

:skull: **Attention** This is under heavy development and should only be run and used by OONI developers. Use at your own risk and if you do use it, you may have to do some cleanup when we release the first public release of OONI Probe.

## Development setup

This project uses [`dep`](https://golang.github.io/dep/) with the `vendor/` dir
in `.gitignore`.

Once you have `dep` installed, run:

```
dep ensure
```

Next, you'll need a recent version of [Measurement Kit](http://github.com/measurement-kit).
As this is a work in progress, you'll likely need to build a version of the
library from source.

You should then be able to build a ooni binary by running:

```
make build
```


If you want to build gooni against a development version of MK without
installing it to your system, you can explicitly specify the path where MK
was built as

```
CGO_LDFLAGS="-L/path/to/measurement-kit/.libs/" CGO_CFLAGS="-I/path/to/measurement-kit/include" make build
```
