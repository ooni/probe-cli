#help:
#help: Runs `go test -short -race $GOLANG_CORE_PACKAGES` inside a subshell.
#help:

(
	run --action setup/psiphon
	run --action setup/go
	run $(command -v go) test -short -race -tags shaping,$OONI_PSIPHON_TAGS $GOLANG_CORE_PACKAGES
)
