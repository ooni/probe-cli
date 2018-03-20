# gooni

An attempt at writing OONI Probe in golang.

This is heavy work in progress.

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

## Todo

* Add support for outputing structured logging messages, while tests are
  running, to be consumed by the desktop app

* Add support for the missing tests, namely:
- HTTP Invalid Request Line
- HTTP Header Field Manipulation
- Facebook Messenger
- Telegram
- WhatsApp
- WebConnectivity

* Fix issue with the informed consent being bypassed on first run

* Finish the config file implementation

* Add support for listing results in the CLI
