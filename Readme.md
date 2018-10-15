# OONI Probe CLI

The next generation OONI Probe Command Line Interface.

## Development setup

Be sure you have golang >= 1.8.

This project uses [`dep`](https://golang.github.io/dep/) with the `vendor/` dir
in `.gitignore`.

Once you have `dep` installed, run:

```
dep ensure
```

Next, you'll need a recent version of [Measurement Kit](http://github.com/measurement-kit).

Building a ooni binary for windows and macOS is currently only supported on a
macOS system.

For building a linux ooni binary, you will need a linux system and follow the
intruction in the linux section.

### macOS

On macOS you can build a windows and macOS ooni binary.

This can be done by running:

```
make download-mk-libs
```

This will download the prebuilt measurement-kit binaries.

Then you can build a macOS build by running:

```
make build
```

And a windows build by running:

```
make build-windows
```

### linux

On linux you can only build a linux ooni binary for amd64.

This can be done by running:

```
make download-mk-libs
```

Then you can build ooni by running:

```
make build
```
