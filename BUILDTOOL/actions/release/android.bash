#help:
#help: Builds oonimkall and miniooni for Android.
#help:

run --action setup/android
run --action setup/go
run --action setup/gomobile
run --action setup/psiphon

run $(command -v gomobile) bind -x -target android -o ./MOBILE/android/oonimkall.aar \
	-tags="$OONI_PSIPHON_TAGS" -ldflags '-s -w' ./pkg/oonimkall

# TODO: archive the framework etc etc
# TODO: build miniooni

run $(command -v go) mod tidy
