#!/bin/bash
set -euxo pipefail

./script/go.bash run ./internal/cmd/qatool \
	-destdir ./internal/minipipeline/testdata/webconnectivity/generated \
	-disable-measure

./script/go.bash run ./internal/cmd/minipipeline \
	-destdir ./internal/minipipeline/testdata/webconnectivity/manual/8844 \
	-measurement ./internal/minipipeline/testdata/webconnectivity/manual/8844/measurement.json

./script/go.bash run ./internal/cmd/minipipeline \
	-destdir ./internal/minipipeline/testdata/webconnectivity/manual/dnsgoogle80 \
	-measurement ./internal/minipipeline/testdata/webconnectivity/manual/dnsgoogle80/measurement.json

./script/go.bash run ./internal/cmd/minipipeline \
	-destdir ./internal/minipipeline/testdata/webconnectivity/manual/firefoxcom \
	-measurement ./internal/minipipeline/testdata/webconnectivity/manual/firefoxcom/measurement.json

./script/go.bash run ./internal/cmd/minipipeline \
	-destdir ./internal/minipipeline/testdata/webconnectivity/manual/issue-2456 \
	-measurement ./internal/minipipeline/testdata/webconnectivity/manual/issue-2456/measurement.json

./script/go.bash run ./internal/cmd/minipipeline \
	-destdir ./internal/minipipeline/testdata/webconnectivity/manual/noipv6 \
	-measurement ./internal/minipipeline/testdata/webconnectivity/manual/noipv6/measurement.json

./script/go.bash run ./internal/cmd/minipipeline \
	-destdir ./internal/minipipeline/testdata/webconnectivity/manual/youtube \
	-measurement ./internal/minipipeline/testdata/webconnectivity/manual/youtube/measurement.json

./script/go.bash run ./internal/cmd/minipipeline \
	-measurement ./internal/cmd/minipipeline/testdata/measurement.json \
	-destdir ./internal/cmd/minipipeline/testdata
