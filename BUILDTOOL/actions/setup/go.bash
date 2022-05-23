#help:
#help: Ensures that we're running the required version of golang and otherwise
#help: installs the required version and adjusts the PATH.
#help:

if [[ $(go version | awk '{print $3'}) != "go${GOLANG_VERSION}" ]]; then
	if [[ ! -x $SDK_BASE_DIR/go${GOLANG_VERSION}/bin/go ]]; then
		run $(command -v go) install golang.org/dl/go${GOLANG_VERSION}@latest
		run $HOME/go/bin/go${GOLANG_VERSION} download
	fi
	run export PATH=$SDK_BASE_DIR/go${GOLANG_VERSION}/bin:$PATH
	if [[ $(go version | awk '{print $3'}) != "go${GOLANG_VERSION}" ]]; then
		fatal "cannot configure go${GOLANG_VERSION}"
	fi
fi
run $(command -v go) version
