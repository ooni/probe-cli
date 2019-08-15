GO ?= GOPATH="" go

install-dev-deps:
	@$(GO) get golang.org/x/tools/cmd/cover
	@$(GO) get github.com/mattn/goveralls

build:
	@echo "Building dist/ooni"
	@$(GO) build -o dist/ooni cmd/ooni/main.go
.PHONY: build

build-windows:
	@echo "Building dist/windows/amd64/ooni.exe"
	@./build.sh windows

build-linux:
	@echo "Building dist/linux/amd64/ooni"
	@./build.sh linux

build-macos:
	@echo "Building dist/macos/amd64/ooni"
	@./build.sh macos

build-all: build-windows build-linux build-macos
.PHONY: build-all build-windows build-linux build-macos

bindata:
	@$(GO) run vendor/github.com/shuLhan/go-bindata/go-bindata/*.go \
		-nometadata	\
		-o internal/bindata/bindata.go -pkg bindata \
	    data/...;

release:
	GITHUB_TOKEN=`cat ~/.GORELEASE_GITHUB_TOKEN` goreleaser --rm-dist
	./build.sh linux
	mv dist/linux/amd64 dist/ooniprobe_$(git describe --tags | sed s/^v//)_linux_amd64
	tar cvzf dist/ooniprobe_$(git describe --tags | sed s/^v//)_linux_amd64.tar.gz \
		-C dist/ooniprobe_$(git describe --tags | sed s/^v//)_linux_amd64 \
		ooni
	cd dist && shasum -a 256 ooniprobe_$(git describe --tags | sed s/^v//)_linux_amd64.tar.gz >> ooniprobe_checksums.txt
	gpg -a --detach-sign dist/ooniprobe_checksums.txt

test-internal:
	@$(GO) test -v ./internal/...

.PHONY: bindata
