#help:
#help: Builds miniooni and ooniprobe for Windows.
#help:

run --action setup/go
run --action setup/mingw
run --action setup/psiphon

(
	run export GOOS=windows GOARCH=386 CGO_ENABLED=1 CC=i686-w64-mingw32-gcc
	run $(command -v go) build -tags="$OONI_PSIPHON_TAGS" -ldflags="-s -w" \
		-o ./CLI/ooniprobe-windows-386.exe ./cmd/ooniprobe
)

(
	run export GOOS=windows GOARCH=386 CGO_ENABLED=1 CC=i686-w64-mingw32-gcc
	run $(command -v go) build -tags="$OONI_PSIPHON_TAGS" -ldflags="-s -w" \
		-o ./CLI/miniooni-windows-386.exe ./internal/cmd/miniooni
)

(
	run export GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc
	run $(command -v go) build -tags="$OONI_PSIPHON_TAGS" -ldflags="-s -w" \
		-o ./CLI/ooniprobe-windows-amd64.exe ./cmd/ooniprobe
)

(
	run export GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc
	run $(command -v go) build -tags="$OONI_PSIPHON_TAGS" -ldflags="-s -w" \
		-o ./CLI/miniooni-windows-amd64.exe ./internal/cmd/miniooni
)

# Is there a mingw-w64 for arm64? For now, let's build without C dependencies
(
	run export GOOS=windows GOARCH=arm64 CGO_ENABLED=0
	run $(command -v go) build -tags="$OONI_PSIPHON_TAGS" -ldflags="-s -w" \
		-o ./CLI/miniooni-windows-arm64.exe ./internal/cmd/miniooni
)
