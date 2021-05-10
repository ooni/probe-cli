#!/bin/sh
command -v make >/dev/null || {
    echo "checking for make... not found"; exit 1; }
exec make -f build.mk "$@"
