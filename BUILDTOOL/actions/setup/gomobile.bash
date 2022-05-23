#help:
#help: Installs the gomobile tool
#help:

run $(command -v go) install golang.org/x/mobile/cmd/gomobile@latest
if [[ -z $(command -v gomobile) ]]; then
	export PATH="$($(command -v go) env GOPATH):$PATH"
fi
run $(command -v gomobile) init
run $(command -v go) get -d golang.org/x/mobile/cmd/gomobile
