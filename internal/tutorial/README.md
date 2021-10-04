# Tutorials: writing OONI nettests

This package contains a living tutorial explaining how to write OONI
nettests. The code in here is based on existing nettests.

Because it's committed to the probe-cli repository and depends on
real OONI code, it should always be up to date.

## Index

- [Rewriting the torsf experiment](experiment/torsf/)

- [Using the measurex package to write network experiments](measurex)

- [Low-level networking using netxlite](netxlite)

## Regenerating the tutorials

```
(cd ./internal/tutorial && go run ./generator)
```
