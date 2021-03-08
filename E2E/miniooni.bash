#!/bin/bash
set -e
go run ./internal/cmd/getresources
go build -v ./internal/cmd/miniooni
probeservices=()
probeservices+=( "https://ps1.ooni.io" )
probeservices+=( "https://dvp6h0xblpcqp.cloudfront.net" )
probeservices+=( "https://ams-pg-test.ooni.org" )
for ps in ${probeservices[@]}; do
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
