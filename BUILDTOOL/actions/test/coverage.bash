#help:
#help: Runs `go test -short -race $GOLANG_CORE_PACKAGES` in a subshell and
#help: saves coverage information at $COVERAGE_REPORT_FILE.
#help:

(
	run --action setup/psiphon
	run --action setup/go
	run $(command -v go) test -short -race -tags shaping,$OONI_PSIPHON_TAGS \
		-coverprofile=$COVERAGE_REPORT_FILE $GOLANG_CORE_PACKAGES
)
