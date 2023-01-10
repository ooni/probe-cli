#doc:
#doc: # MONOREPO/tools/libcore.bash
#doc:
#doc: Library of useful bash functions.

#doc:
#doc: ## goos (function)
#doc:
#doc: Echoes the operating system name using Go conventions.
goos() {
	case $(uname -s) in
	Linux)
		echo "linux"
		;;
	Darwin)
		echo "darwin"
		;;
	*)
		echo "FATAL: unsupported system" 1>&2
		exit 1
		;;
	esac
}

#doc:
#doc: ## goarch (function)
#doc:
#doc: Echoes the processor architecture using Go conventions.
goarch() {
	case $(uname -m) in
	amd64 | x86_64)
		echo "amd64"
		;;
	arm64 | aarch64)
		echo "arm64"
		;;
	*)
		echo "FATAL: unsupported arch" 1>&2
		exit 1
		;;
	esac
}

#doc:
#doc: ## fatal (function)
#doc:
#doc: This function prints its arguments on the stderr
#doc: and then terminates the execution.
fatal() {
	echo "🚨 $@" 1>&2
	exit 1
}

#doc:
#doc: ## warn (function)
#doc:
#doc: This function prints a warning message
#doc: on the standard error.
warn() {
	echo "🚨 $@" 1>&2
}

#doc:
#doc: ## info (function)
#doc:
#doc: This function prints an informational message
#doc: on the standard error.
info() {
	echo "🗒️ $@" 1>&2
}

#doc:
#doc: ## success (function)
#doc:
#doc: This function prints a message on the standard
#doc: error indicating that some check succeded.
success() {
	echo "✔️ $@" 1>&2
}

#doc:
#doc: ## run (function)
#doc:
#doc: This function logs a command it's about to
#doc: execute and then executes it.
run() {
	echo "🐚 $@" 1>&2
	"$@"
}
