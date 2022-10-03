#!/bin/bash

# This script checks whether we're able to submit measurements to
# different backends using miniooni. It fails if we cannot find in
# the specific backend the measurement we submitted.
#
# Note: using --tunnel=psiphon assumes that we have been compiling
# miniooni with builtin support for psiphon.

set -euxo pipefail

rm -f E2E/o.jsonl

miniooni="${1:-./miniooni}"

$miniooni --yes -o E2E/o.jsonl \
	--probe-services=https://ams-pg-test.ooni.org/ \
	--tunnel=none \
	web_connectivity -i https://mail.google.com/robots.txt

$miniooni --yes -o E2E/o.jsonl \
	--probe-services=https://dvp6h0xblpcqp.cloudfront.net/ \
	--tunnel=none \
	web_connectivity -i https://mail.google.com/robots.txt

$miniooni --yes -o E2E/o.jsonl \
	--probe-services=https://ams-pg-test.ooni.org/ \
	--tunnel=tor \
	web_connectivity -i https://mail.google.com/robots.txt

$miniooni --yes -o E2E/o.jsonl \
	--probe-services=https://ams-pg-test.ooni.org/ \
	--tunnel=psiphon \
	web_connectivity -i https://mail.google.com/robots.txt

$miniooni --yes -o E2E/o.jsonl \
	--probe-services=https://ams-pg-test.ooni.org/ \
	--tunnel=torsf \
	web_connectivity -i https://mail.google.com/robots.txt

go run ./internal/cmd/e2epostprocess -expected 5 -backend https://ams-pg-test.ooni.org/
