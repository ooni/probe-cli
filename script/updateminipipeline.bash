#!/bin/bash
set -euxo pipefail

go run ./internal/cmd/qatool \
	-destdir ./internal/minipipeline/testdata/webconnectivity/generated \
	-disable-measure

go run ./internal/cmd/minipipeline \
	-destdir ./internal/minipipeline/testdata/webconnectivity/manual/dnsgoogle80 \
	-measurement ./internal/minipipeline/testdata/webconnectivity/manual/dnsgoogle80/measurement.json

go run ./internal/cmd/minipipeline \
	-destdir ./internal/minipipeline/testdata/webconnectivity/manual/noipv6 \
	-measurement ./internal/minipipeline/testdata/webconnectivity/manual/noipv6/measurement.json

go run ./internal/cmd/minipipeline \
	-destdir ./internal/minipipeline/testdata/webconnectivity/manual/youtube \
	-measurement ./internal/minipipeline/testdata/webconnectivity/manual/youtube/measurement.json

go run ./internal/cmd/minipipeline \
	-destdir ./internal/minipipeline/testdata/webconnectivity/manual/issue-2456 \
	-measurement ./internal/minipipeline/testdata/webconnectivity/manual/issue-2456/measurement.json

go run ./internal/cmd/minipipeline \
	-measurement ./internal/cmd/minipipeline/testdata/measurement.json \
	-destdir ./internal/cmd/minipipeline/testdata
