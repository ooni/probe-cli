# Directory GOCACHE

This directory contains the GOCACHE and GOMODCACHE we use when
statically compiling Linux binaries using Docker.

If you keep the content of this directory, subsequent builds will be
faster. You will notice this especially for builds using qemu-user-static
to build for different architectures.
