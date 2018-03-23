GO ?= go

build:
	@echo "Building ./ooni"
	@$(GO) build -i -o dist/ooni cmd/ooni/main.go
.PHONY: build

update-mk:
	@echo "updating mk"
	@dep ensure -update github.com/measurement-kit/go-measurement-kit
.PHONY: update-mk

bindata:
	@$(GO) run vendor/github.com/shuLhan/go-bindata/go-bindata/*.go \
		-nometadata	\
		-o internal/bindata/bindata.go -pkg bindata \
	    data/...;
.PHONY: bindata
