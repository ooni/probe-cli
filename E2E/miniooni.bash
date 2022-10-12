#!/bin/bash

# This script checks whether we're able to submit measurements to
# different backends using miniooni. It fails if we cannot find in
# the specific backend the measurement we submitted.
#
# Note: using --tunnel=psiphon assumes that we have been compiling
# miniooni with builtin support for psiphon.
#
# Note about cloudfront: as of 2022-10-08, dvp6h0xblpcqp.cloudfront.net
# and dkyhjv0wpi2dk.cloudfront.net front distinct aliases of the
# same backend host (backend-fsn.ooni.org). We can use either addr
# and the result should be the same. So, let us test that.

set -euxo pipefail

rm -f E2E/o.jsonl

miniooni="${1:-./miniooni}"

$miniooni --yes -o E2E/o.jsonl \
	--probe-services=https://api.ooni.io/ \
	--tunnel=none \
	web_connectivity -i https://mail.google.com/robots.txt

$miniooni --yes -o E2E/o.jsonl \
	--probe-services=https://dvp6h0xblpcqp.cloudfront.net/ \
	--tunnel=none \
	web_connectivity -i https://mail.google.com/robots.txt

$miniooni --yes -o E2E/o.jsonl \
	--probe-services=https://dkyhjv0wpi2dk.cloudfront.net/ \
	--tunnel=none \
	web_connectivity -i https://mail.google.com/robots.txt

$miniooni --yes -o E2E/o.jsonl \
	--probe-services=https://api.ooni.io/ \
	--tunnel=tor \
	web_connectivity -i https://mail.google.com/robots.txt

$miniooni --yes -o E2E/o.jsonl \
	--probe-services=https://api.ooni.io/ \
	--tunnel=psiphon \
	web_connectivity -i https://mail.google.com/robots.txt

$miniooni --yes -o E2E/o.jsonl \
	--probe-services=https://api.ooni.io/ \
	--tunnel=torsf \
	web_connectivity -i https://mail.google.com/robots.txt

go run ./internal/cmd/e2epostprocess -expected 6 -backend https://api.ooni.io/
