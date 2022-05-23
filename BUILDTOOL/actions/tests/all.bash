#help:
#help: Runs `go test -race ./...` inside a subshell.
#help:

(
	run --action setup/psiphon
	run --action setup/go
	run $(command -v go) test -race -tags shaping,$OONI_PSIPHON_TAGS ./...
)
