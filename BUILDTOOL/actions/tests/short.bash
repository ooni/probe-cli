#help:
#help: Runs `go test -short -race ./...` inside a subshell.
#help:

(
	run --action setup/psiphon
	run --action setup/go
	run $(command -v go) test -short -race -tags shaping,$OONI_PSIPHON_TAGS ./...
)
