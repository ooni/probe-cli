# Tutorials: writing OONI nettests

This package contains a living tutorial explaining how to write OONI
nettests. The code in here is based on existing nettests.

Because it's committed to the probe-cli repository and depends on
real OONI code, it should always be up to date.

## Index

- [Rewriting the torsf experiment](experiment/torsf/)

- [Performing measurements using internal/measure](measure)

## Regenerating the tutorials

Most of the text of these tutorials comes from comments in real
Go code, to ensure that the code we show is always working against
the main development branch. For this reason, one should not edit
the README.md files manually when a Go file is also present in the
same directory. The following command regenerates all tutorials.

```
(cd ./internal/tutorial && go run ./generator)
```