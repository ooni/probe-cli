GO ?= go

build:
	@echo "Building dist/ooni"
	@$(GO) build -i -o dist/ooni cmd/ooni/main.go
.PHONY: build

build-windows:
	@echo "Building dist/ooni.exe"
	CC=x86_64-w64-mingw32-gcc GOOS=windows GOARCH=amd64 CGO_ENABLED=1 go build -o dist/ooni.exe -x cmd/ooni/main.go

update-mk:
	@echo "updating mk"
	@dep ensure -update github.com/measurement-kit/go-measurement-kit
	@test -f vendor/github.com/measurement-kit/go-measurement-kit/libs/ || cd vendor && git submodule add https://github.com/measurement-kit/golang-prebuilt.git github.com/measurement-kit/go-measurement-kit/libs # This is a hack to workaround: https://github.com/golang/dep/issues/1240
.PHONY: update-mk

bindata:
	@$(GO) run vendor/github.com/shuLhan/go-bindata/go-bindata/*.go \
		-nometadata	\
		-o internal/bindata/bindata.go -pkg bindata \
	    data/...;
.PHONY: bindata
