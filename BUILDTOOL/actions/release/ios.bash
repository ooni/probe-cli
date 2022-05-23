#help:
#help: Builds oonimkall for iOS.
#help:

run --action setup/go
run --action setup/gomobile
run --action setup/psiphon
run $(command -v gomobile) bind -x -target ios -o ./MOBILE/ios/oonimkall.xcframework \
	-tags="$OONI_PSIPHON_TAGS" -ldflags '-s -w' ./pkg/oonimkall

# TODO: setup iOS
# TODO: archive the framework etc etc

run $(command -v go) mod tidy
