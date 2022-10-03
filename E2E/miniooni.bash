#!/bin/bash

# This script checks whether we're able to submit measurements to
# different backends using miniooni. It fails if we cannot find in
# the specific backend the measurement we submitted.
#
# Note: using --tunnel=psiphon assumes that we have been compiling
# miniooni with builtin support for psiphon.

set -euxo pipefail

backends=()
backends+=( "https://api.ooni.io" )
backends+=( "https://dvp6h0xblpcqp.cloudfront.net" )
backends+=( "https://ams-pg-test.ooni.org" )

miniooni="${1:-./miniooni}"
for ps in ${backends[@]}; do
    opt="-o E2E/o.jsonl --probe-services=$ps"
    $miniooni --yes $opt -i http://mail.google.com web_connectivity
done

$miniooni --tunnel=psiphon --yes -i http://mail.google.com web_connectivity
$miniooni --tunnel=tor --yes -i http://mail.google.com web_connectivity

#go run ./internal/cmd/e2epostprocess -expected 5  # TODO(bassosimone): fix this
