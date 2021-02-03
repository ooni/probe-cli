# Package github.com/ooni/probe-cli/internal/libminiooni

Package libminiooni implements the cmd/miniooni CLI. Miniooni is our
experimental client used for research and QA testing.

This CLI has CLI options that do not conflict with Measurement Kit
v0.10.x CLI options. There are some options conflict with the legacy
OONI Probe CLI options. Perfect backwards compatibility is not a
design goal for miniooni. Rather, we aim to have as little conflict
as possible such that we can run side by side QA checks.

This package was split off from cmd/miniooni in ooni/probe-engine. For
now we are keeping this split, but we will merge them in the future.
