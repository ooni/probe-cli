GO_BINDATA_VERSION = $(shell go-bindata --version | cut -d ' ' -f2 | head -n 1 || echo "missing")
REQ_GO_BINDATA_VERSION = 3.2.0
GO ?= go

build:
	@echo "Building ./ooni"
	@$(GO) build -o ooni cmd/ooni/main.go
.PHONY: build

bindata:
ifneq ($(GO_BINDATA_VERSION),$(REQ_GO_BINDATA_VERSION))
	go get -u github.com/shuLhan/go-bindata/...;
endif
	@go-bindata \
		-nometadata	\
		-o internal/bindata/bindata.go -pkg bindata \
	    data;
.PHONY: bindata
