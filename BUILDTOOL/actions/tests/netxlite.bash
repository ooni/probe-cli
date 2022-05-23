#help:
#help: Runs `go test -race ./internal/netxlite/...` inside a subshell.
#help:

(
	run --action setup/go
	run $(command -v go) test -race ./internal/netxlite/...
)
