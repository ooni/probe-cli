#!/bin/bash
set -euo pipefail
exitcode=0
for file in $(find . -type f -name \*.go); do
	if [ "$file" = "./internal/netxlite/iox.go" ]; then
		# We're allowed to use ReadAll and Copy in this file to
		# implement safer wrappers for these functions.
		continue
	fi
	if [ "$file" = "./internal/netxlite/filtering/http.go" ]; then
		# We're allowed to use ReadAll and Copy in this file to
		# avoid depending on netxlite, so we can use filtering
		# inside of netxlite's own test suite.
		continue
	fi
	if grep -q 'io\.ReadAll' $file; then
		echo "in $file: do not use io.ReadAll, use netxlite.ReadAllContext" 1>&2
		exitcode=1
	fi
	if grep -q 'ioutil\.ReadAll' $file; then
		echo "in $file: do not use ioutil.ReadAll, use netxlite.ReadAllContext" 1>&2
		exitcode=1
	fi
	if grep -q 'io\.Copy' $file; then
		echo "in $file: do not use io.Copy, use netxlite.CopyContext" 1>&2
		exitcode=1
	fi
	if grep -q 'ioutil\.Copy' $file; then
		echo "in $file: do not use ioutil.Copy, use netxlite.CopyContext" 1>&2
		exitcode=1
	fi
done
exit $exitcode
