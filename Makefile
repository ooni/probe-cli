GO ?= go

build:
	@echo "Building ./ooni"
	@$(GO) build -o ooni cmd/ooni/main.go
.PHONY: build
