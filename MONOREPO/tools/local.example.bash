#doc:
#doc: # MONOREPO/tools/local.example.bash
#doc:
#doc: Example of a possible local.bash file. By copying this file
#doc: to MONOREPO/tools/local.bash, you are able to override the
#doc: repositories tracked by the monorepo scripts.

repositories=(
	. # the dot is git@github.com:ooni/probe-cli and MUST be first
	git@github.com:ooni/probe-android
	git@github.com:ooni/probe-desktop
	git@github.com:ooni/probe-ios
	git@github.com:ooni/spec
)
