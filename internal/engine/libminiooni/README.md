# Package github.com/ooni/probe-engine/libminiooni

Package libminiooni implements the cmd/miniooni CLI. Miniooni is our
experimental client used for research and QA testing.

This CLI has CLI options that do not conflict with Measurement Kit
v0.10.x CLI options. There are some options conflict with the legacy
OONI Probe CLI options. Perfect backwards compatibility is not a
design goal for miniooni. Rather, we aim to have as little conflict
as possible such that we can run side by side QA checks.

We extracted this package from cmd/miniooni to allow us to further
integrate the miniooni CLI into other binaries (see for example the
code at github.com/bassosimone/aladdin).
