#!/bin/bash
set -euo pipefail

#help:
#help: Usage: ./BUILDTOOL/main.bash [-h|--help|h|help]
#help:        ./BUILDTOOL/main.bash lt|list-targets
#help:        ./BUILDTOOL/main.bash sc|show-config
#help:        ./BUILDTOOL/main.bash d|describe <target>
#help:        ./BUILDTOOL/main.bash r|run <target>
#help:
#help: The first form of the command prints this help message.
#help:
#help: The second form of the command lists all the available targets.
#help:
#help: The third form of the command shows the config.
#help:
#help: The fourth form of the command describes a given target.
#help:
#help: The fifth form of the command runs the given target.
#help:

#config: ANDROID_BUILDTOOLS_VERSION
ANDROID_BUILDTOOLS_VERSION=32.0.0

#config: ANDROID_CLITOOLS_VERSION
ANDROID_CLITOOLS_VERSION=8512546

#config: ANDROID_PLATFORM_VERSION
ANDROID_PLATFORM_VERSION=android-31

#config: GOLANG_VERSION
GOLANG_VERSION=1.18.2

#config: SDK_BASE_DIR
SDK_BASE_DIR=$HOME/sdk

#config: GOLANG_DOCKER_IMAGE
GOLANG_DOCKER_IMAGE=golang:${GOLANG_VERSION}-alpine

#config: ANDROID_CLITOOLS_SHA256
ANDROID_CLITOOLS_SHA256=5e7bf2dd563d34917d32f3c5920a85562a795c93

#config: ANDROID_NDK_VERSION
ANDROID_NDK_VERSION=23.1.7779620

#config: ANDROID_SDK_DIR
ANDROID_SDK_DIR=$SDK_BASE_DIR/ooni-android

#config: COVERAGE_REPORT_FILE
COVERAGE_REPORT_FILE=probe-cli.cov

fatal() {
	echo "🚨 $@" 1>&2
	exit 1
}

run() {
	if [[ $# > 1 && $1 == "--action" ]]; then
		shift
		local name=./BUILDTOOL/actions/$1.bash
		shift
		if [[ ! -f $name ]]; then
			fatal "no such action: $name"
		fi
		source $name
	else
		echo "🚧🐙 $@" 1>&2
		"$@"
	fi
}

showhelp() {
	cat $(readlink -f $0) | grep '^#help:' | sed -e 's/^#help://g'
}

if [[ $# -le 0 || $1 == "-h" || $1 == "--help" || $1 == "h" || $1 == "help" ]]; then
	showhelp
	exit 0
fi

if [[ ($1 == "lt" || $1 == "list-targets") && $# -eq 1 ]]; then
	(
		prefix=./BUILDTOOL/actions
		for filename in $(find $prefix -type f -name \*.bash | sort); do
			name=${filename#$prefix/}
			name=${name%.bash}
			echo "- $name"
		done
	)
	exit 0
fi

if [[ ($1 == "sc" || $1 == "show-config") && $# -eq 1 ]]; then
	(
		echo ""
		for name in $(cat $(readlink -f $0) | grep '^#config: ' | sed -e 's/^#config://g' | sort); do
			echo "$name=${!name}"
			echo ""
		done
	)
	exit 0
fi

if [[ ($1 == "d" || $1 == "describe") && $# -eq 2 ]]; then
	(
		filename=./BUILDTOOL/actions/$2.bash
		if [[ ! -f $filename ]]; then
			fatal "no such target: $2"
		fi
		echo ""
		echo "$2 ($filename)"
		cat $filename | grep '^#help:' | sed -e 's/^#help://g'
	)
	exit 0
fi

if [[ ($1 == "r" || $1 == "run") && $# -eq 2 ]]; then
	run mkdir -p $SDK_BASE_DIR
	run --action $2
	exit 0
fi

fatal "invalid command line invocation (try: '$0 -h' for more help)"
