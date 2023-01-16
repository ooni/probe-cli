#!/bin/bash
set -euo pipefail

if [[ $# -ne 2 ]]; then
	echo "usage: $0 {name-before} {name-after}" 1>&2
	exit 1
fi
name_before=$1
shift
name_after=$1
shift

basename_before=$(basename $name_before)
basename_after=$(basename $name_after)

git mv $name_before $name_after

for file in $(find $name_after -type f -name \*.go); do
	cat $file | sed -e "s|^package $basename_before|package $basename_after|g" \
		-e "s|^// Package $basename_before|// Package $basename_after|g" > $file.tmp
	cat $file.tmp > $file
	rm $file.tmp
done

pkg_prefix=github.com/ooni/probe-cli/v3
pkg_before=$pkg_prefix/$name_before
pkg_after=$pkg_prefix/$name_after

for file in $(find . -type f -name \*.go); do
	cat $file | sed -e "s|\"$pkg_before\"|\"$pkg_after\"|g" > $file.tmp
	cat $file.tmp > $file
	rm $file.tmp
done
