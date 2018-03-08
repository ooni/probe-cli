GO ?= go

build:
	@echo "Building ./ooni"
	@$(GO) build -i -o ooni cmd/ooni/main.go
.PHONY: build

bindata:
	@$(GO) run vendor/github.com/shuLhan/go-bindata/go-bindata/*.go \
		-nometadata	\
		-o internal/bindata/bindata.go -pkg bindata \
	    data/...;
.PHONY: bindata
