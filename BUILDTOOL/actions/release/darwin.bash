#help:
#help: Builds miniooni and ooniprobe for darwin (aka macOS).
#help:

# TODO(bassosimone): check for the presence of CLI tools

run --action setup/go
run --action setup/psiphon

(
	run export GOOS=darwin GOARCH=amd64 CGO_ENABLED=1
	run $(command -v go) build -tags="$OONI_PSIPHON_TAGS" -ldflags="-s -w" \
		-o ./CLI/ooniprobe-darwin-amd64 ./cmd/ooniprobe
)

(
	run export GOOS=darwin GOARCH=amd64 CGO_ENABLED=1
	run $(command -v go) build -tags="$OONI_PSIPHON_TAGS" -ldflags="-s -w" \
		-o ./CLI/miniooni-darwin-amd64 ./internal/cmd/miniooni
)

(
	run export GOOS=darwin GOARCH=arm64 CGO_ENABLED=1
	run $(command -v go) build -tags="$OONI_PSIPHON_TAGS" -ldflags="-s -w" \
		-o ./CLI/ooniprobe-darwin-arm64 ./cmd/ooniprobe
)

(
	run export GOOS=darwin GOARCH=arm64 CGO_ENABLED=1
	run $(command -v go) build -tags="$OONI_PSIPHON_TAGS" -ldflags="-s -w" \
		-o ./CLI/miniooni-darwin-arm64 ./internal/cmd/miniooni
)
