#help:
#help: Builds ooniprobe and miniooni for Linux. We build miniooni to use the Go
#help: internal resolver written in Go and to not depend on libc. We build
#help: ooniprobe inside an Alpine container and statically link with musl libc.
#help:

run --action setup/go
run --action setup/psiphon

# TODO: check for docker support

(
	run export GOOS=linux GOARCH=386 CGO_ENABLED=0
	run $(command -v go) build -tags="netgo,$OONI_PSIPHON_TAGS" \
		-ldflags="-s -w -extldflags -static" \
		-o ./CLI/miniooni-linux-386 ./internal/cmd/miniooni
)

(
	run export GOOS=linux GOARCH=amd64 CGO_ENABLED=0
	run $(command -v go) build -tags="netgo,$OONI_PSIPHON_TAGS" \
		-ldflags="-s -w -extldflags -static" \
		-o ./CLI/miniooni-linux-amd64 ./internal/cmd/miniooni
)

(
	run export GOOS=linux GOARCH=arm CGO_ENABLED=0 GOARM=6
	run $(command -v go) build -tags="netgo,$OONI_PSIPHON_TAGS" \
		-ldflags="-s -w -extldflags -static" \
		-o ./CLI/miniooni-linux-armv6 ./internal/cmd/miniooni
)

(
	run export GOOS=linux GOARCH=arm CGO_ENABLED=0 GOARM=7
	run $(command -v go) build -tags="netgo,$OONI_PSIPHON_TAGS" \
		-ldflags="-s -w -extldflags -static" \
		-o ./CLI/miniooni-linux-armv7 ./internal/cmd/miniooni
)

(
	run export GOOS=linux GOARCH=arm64 CGO_ENABLED=0
	run $(command -v go) build -tags="netgo,$OONI_PSIPHON_TAGS" \
		-ldflags="-s -w -extldflags -static" \
		-o ./CLI/miniooni-linux-arm64 ./internal/cmd/miniooni
)

(
	run $(command -v docker) pull --platform linux/386 $GOLANG_DOCKER_IMAGE
	run $(command -v docker) run --platform linux/386 -e GOARCH=386 -v $(pwd):/ooni -w /ooni \
		$GOLANG_DOCKER_IMAGE ./CLI/build-linux -tags=$OONI_PSIPHON_TAGS
)

(
	run $(command -v docker) pull --platform linux/amd64 $GOLANG_DOCKER_IMAGE
	run $(command -v docker) run --platform linux/amd64 -e GOARCH=amd64 -v $(pwd):/ooni -w /ooni \
		$GOLANG_DOCKER_IMAGE ./CLI/build-linux -tags=$OONI_PSIPHON_TAGS
)

(
	run $(command -v docker) pull --platform linux/arm/v6 $GOLANG_DOCKER_IMAGE
	run $(command -v docker) run --platform linux/arm/v6 -e GOARCH=arm -e GOARM=6 \
		-v $(pwd):/ooni -w /ooni $GOLANG_DOCKER_IMAGE \
		./CLI/build-linux -tags=$OONI_PSIPHON_TAGS
)

(
	run $(command -v docker) pull --platform linux/arm/v7 $GOLANG_DOCKER_IMAGE
	run $(command -v docker) run --platform linux/arm/v7 -e GOARCH=arm -e GOARM=7 \
		-v $(pwd):/ooni -w /ooni $GOLANG_DOCKER_IMAGE \
		./CLI/build-linux -tags=$OONI_PSIPHON_TAGS
)

(
	run $(command -v docker) pull --platform linux/arm64 $GOLANG_DOCKER_IMAGE
	run $(command -v docker) run --platform linux/arm64 -e GOARCH=arm64 -v $(pwd):/ooni -w /ooni \
		$GOLANG_DOCKER_IMAGE ./CLI/build-linux -tags=$OONI_PSIPHON_TAGS
)
