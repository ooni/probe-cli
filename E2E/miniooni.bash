#!/bin/bash
#
# This script checks whether we're able to submit measurements to
# different backends using miniooni. It fails if we cannot find in
# the specific backend the measurement we submitted.
#
set -e
backends=()
backends+=( "https://ps1.ooni.io" )
backends+=( "https://dvp6h0xblpcqp.cloudfront.net" )
backends+=( "https://ams-pg-test.ooni.org" )
for ps in ${backends[@]}; do
    opt="-o E2E/o.jsonl --probe-services=$ps"
    set -x
    ./miniooni --yes $opt -i http://mail.google.com web_connectivity
    ./miniooni --yes $opt tor
    ./miniooni --yes $opt psiphon
    set +x
done
set -x
go run ./internal/cmd/e2epostprocess -expected 9
set +x
