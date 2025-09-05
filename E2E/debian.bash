#!/bin/bash

# This script checks whether we can install ooniprobe on debian
# for a specific architecture using docker and the official
# install instructions published at https://ooni.org/install.

set -euo pipefail

install_flow() {
	set -x
	export DEBIAN_FRONTEND=noninteractive
	dpkg --add-architecture "$1"
	apt-get update
	apt-get install --yes gnupg
	wget -O- https://ooni.org/ooniprobe.asc | gpg --dearmor | tee /usr/share/keyrings/ooniprobe-archive-keyring.gpg > /dev/null
	echo "deb [arch=$1 signed-by=/usr/share/keyrings/ooniprobe-archive-keyring.gpg] http://deb.ooni.org/ unstable main" \
        | tee /etc/apt/sources.list.d/ooniprobe.list
	apt-get update
	apt-get install --yes ooniprobe-cli
	dpkg -l | grep ooniprobe-cli > DEBIAN_INSTALLED_PACKAGE.txt
}

docker_flow() {
	printf "checking for docker..."
	command -v docker || {
		echo "not found"
		exit 1
	}
	set -x
	docker pull debian:stable
	docker run -v "$(pwd):/ooni" -w /ooni debian:stable ./E2E/debian.bash install "$1"
}

if [ "$1" = "docker" ]; then
	test -n "$2" || {
		echo "usage: $0 docker {i386,amd64,armhf,arm64}" 1>&2
		exit 1
	}
	docker_flow "$2"

elif [ "$1" = "install" ]; then
	install_flow "$2"

else
	echo "usage: $0 docker {i386,amd64,armhf,arm64}" 1>&2
	echo "       $0 install {i386,amd64,armhf,arm64}" 1>&2
	exit 1
fi
