#!/bin/sh
# This script checks whether we can install ooniprobe on debian
# for a specific architecture using docker and the official
# install instructions published at https://ooni.org/install.

set -e

install_flow() {
	set -x
	export DEBIAN_FRONTEND=noninteractive
	apt-get update
	apt-get install --yes gnupg
	apt-key adv --verbose --keyserver hkp://keyserver.ubuntu.com --recv-keys 'B5A08F01796E7F521861B449372D1FF271F2DD50'
	echo "deb [arch=$1] http://deb.ooni.org/ unstable main" | tee /etc/apt/sources.list.d/ooniprobe.list
	apt-get update
	apt-get install --yes ooniprobe-cli
}

docker_flow() {
	printf "checking for docker..."
	command -v docker || {
		echo "not found"
		exit 1
	}
	set -x
	docker pull --platform "linux/$1" debian:stable
	docker run --platform "linux/$1" -v "$(pwd):/ooni" -w /ooni debian:stable ./E2E/debian.sh install
}

if [ "$1" = "docker" ]; then
	test -n "$2" || {
		echo "usage: $0 docker {i386,amd64,armhf,arm64}" 1>&2
		exit 1
	}
	docker_flow "$2"

elif [ "$1" = "install" ]; then
	install_flow $1

else
	echo "usage: $0 docker {i386,amd64,armhf,arm64}" 1>&2
	echo "       $0 install" 1>&2
	exit 1
fi
