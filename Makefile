#
# help
#

.PHONY: help # prints this help screen
help:
	@echo "Available targets:"
	@cat Makefile|grep '^.PHONY:'|sed -e 's/^\.PHONY: /- /g' -e 's/ #/:/g'

#
# GOSDK
#

# GOVERSION is the version of Go we use.
GOVERSION = 1.16.4

# GOPATH is the path where Go installs binaries.
GOPATH = `go env GOPATH`

# GOSDK is the Go SDK we use.
GOSDK = $(HOME)/sdk/go$(GOVERSION)

# $(GOSDK) ensures we have a Go SDK.
$(GOSDK):
	go get $(GOOPTIONS) golang.org/dl/go$(GOVERSION)
	$(GOPATH)/bin/go$(GOVERSION) download

#
# miniooni
#

# CLI/linux/amd64/miniooni is miniooni for linux/amd64.
CLI/linux/amd64/miniooni: $(GOSDK)
	PATH=$(GOSDK)/bin:$(PATH) GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build $(GOOPTIONS) -o $@ ./internal/cmd/miniooni

# CLI/linux/386/miniooni is miniooni for linux/386.
CLI/linux/386/miniooni: $(GOSDK)
	PATH=$(GOSDK)/bin:$(PATH) GOOS=linux GOARCH=386 CGO_ENABLED=0 go build $(GOOPTIONS) -o $@ ./internal/cmd/miniooni

# CLI/linux/arm64/miniooni is miniooni for linux/arm64.
CLI/linux/arm64/miniooni: $(GOSDK)
	PATH=$(GOSDK)/bin:$(PATH) GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build $(GOOPTIONS) -o $@ ./internal/cmd/miniooni

# CLI/linux/arm/miniooni is miniooni for linux/arm.
CLI/linux/arm/miniooni: $(GOSDK)
	PATH=$(GOSDK)/bin:$(PATH) GOOS=linux GOARCH=arm CGO_ENABLED=0 go build $(GOOPTIONS) -o $@ ./internal/cmd/miniooni

# CLI/darwin/amd64/miniooni is miniooni for darwin/amd64.
CLI/darwin/amd64/miniooni: $(GOSDK)
	PATH=$(GOSDK)/bin:$(PATH) GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build $(GOOPTIONS) -o $@ ./internal/cmd/miniooni

# CLI/darwin/arm64/miniooni is miniooni for darwin/arm64.
CLI/darwin/arm64/miniooni: $(GOSDK)
	PATH=$(GOSDK)/bin:$(PATH) GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build $(GOOPTIONS) -o $@ ./internal/cmd/miniooni

# CLI/windows/386/miniooni is miniooni for windows/386.
CLI/windows/386/miniooni: $(GOSDK)
	PATH=$(GOSDK)/bin:$(PATH) GOOS=windows GOARCH=386 CGO_ENABLED=0 go build $(GOOPTIONS) -o $@ ./internal/cmd/miniooni

# CLI/windows/amd64/miniooni is miniooni for windows/amd64.
CLI/windows/amd64/miniooni: $(GOSDK)
	PATH=$(GOSDK)/bin:$(PATH) GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build $(GOOPTIONS) -o $@ ./internal/cmd/miniooni

MINIOONI_TARGETS = CLI/linux/amd64/miniooni   \
		   CLI/linux/386/miniooni     \
		   CLI/linux/arm64/miniooni   \
		   CLI/linux/arm/miniooni     \
		   CLI/darwin/amd64/miniooni  \
		   CLI/darwin/arm64/miniooni  \
		   CLI/windows/386/miniooni   \
		   CLI/windows/amd64/miniooni

.PHONY: miniooni # builds miniooni for every available GOOS and GOARCH
miniooni: $(MINIOONI_TARGETS)

#
# ooniprobe
#

.PHONY: ooniprobe/windows # builds ooniprobe for Windows
ooniprobe/windows:

.PHONY: ooniprobe/linux # builds ooniprobe for Linux
ooniprobe/linux:

.PHONY: ooniprobe/darwin # builds ooniprobe for Darwin/macOS
ooniprobe/darwin:

#
# oonimkall/android
#

.PHONY: oonimkall/android # builds the oonimkall library for Android
oonimkall/android:

#
# oonimkall/ios
#

.PHONY: oonimkall/ios # builds the oonimkall library for iOS
oonimkall/ios:

