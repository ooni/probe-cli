#!/bin/bash

#
# This script uses cURL to verify that Jafar is able to produce a
# bunch of censorship conditions. It should be noted that this script
# only works on Linux and will never work on other systems.
#

set -e

function execute() {
  echo "+ $@" 1>&2
  "$@"
}

function expectexitcode() {
  local expect
  local exitcode
  expect=$1
  shift
  set +e
  "$@"
  exitcode=$?
  set -e
  echo "expected exitcode $expect, found $exitcode" 1>&2
  if [ $exitcode != $expect ]; then
    exit 1
  fi
}

function runtest() {
  echo "=== BEGIN $1 ==="
  "$1"
  echo "=== END $1 ==="
}

function http_got_nothing() {
  expectexitcode 52 execute ./jafar -iptables-hijack-http-to 127.0.0.1:7117  \
    -main-command 'curl -sm5 --connect-to ::example.com: http://ooni.io'
}

function http_recv_error() {
  expectexitcode 56 execute ./jafar -iptables-reset-keyword ooni          \
    -main-command 'curl -sm5 --connect-to ::example.com: http://ooni.io'
}

function http_operation_timedout() {
  expectexitcode 28 execute ./jafar -iptables-drop-keyword ooni           \
    -main-command 'curl -sm5 --connect-to ::example.com: http://ooni.io'
}

function http_couldnt_connect() {
  local ip
  ip=$(host -tA example.com|cut -f4 -d' ')
  expectexitcode 7 execute ./jafar -iptables-reset-ip $ip                 \
    -main-command 'curl -sm5 --connect-to ::example.com: http://ooni.io'
}

function http_blockpage() {
  outfile=$(mktemp)
  chown nobody $outfile  # curl runs as user nobody
  expectexitcode 0 execute ./jafar -http-proxy-block ooni  \
    -iptables-hijack-http-to 127.0.0.1:80                  \
    -main-command "curl -so $outfile --connect-to ::example.com: http://ooni.io"
  if ! grep -q '451 Unavailable For Legal Reasons' $outfile; then
    echo "fatal: the blockpage does not contain the expected pattern" 1>&2
    exit 1
  fi
}

function dns_injection() {
  output=$(expectexitcode 0 execute ./jafar                              \
    -iptables-hijack-dns-to 127.0.0.1:53                                 \
    -dns-proxy-hijack ooni                                               \
    -main-command 'dig +time=2 +short @example.com ooni.io')
  if [ "$output" != "127.0.0.1" ]; then
    echo "fatal: the resulting IP is not the expected one" 1>&2
    exit 1
  fi
}

function dns_timeout() {
  expectexitcode 9 execute ./jafar             \
    -iptables-hijack-dns-to 127.0.0.1:53       \
    -dns-proxy-ignore ooni                     \
    -main-command 'dig +time=2 +short @example.com ooni.io'
}

function dns_nxdomain() {
  output=$(expectexitcode 0 execute ./jafar                              \
    -iptables-hijack-dns-to 127.0.0.1:53                                 \
    -dns-proxy-block ooni                                                \
    -main-command 'dig +time=2 +short @example.com ooni.io')
  if [ "$output" != "" ]; then
    echo "fatal: expected no output here" 1>&2
    exit 1
  fi
}

function sni_man_in_the_middle() {
  expectexitcode 60 execute ./jafar -iptables-hijack-https-to 127.0.0.1:4114  \
    -main-command 'curl -sm5 --connect-to ::example.com: https://ooni.io'
}

function sni_got_nothing() {
  expectexitcode 52 execute ./jafar -iptables-hijack-https-to 127.0.0.1:4114  \
    -main-command 'curl -sm5 --cacert badproxy.pem --connect-to ::example.com: https://ooni.io'
}

function sni_connect_error() {
  expectexitcode 35 execute ./jafar -iptables-reset-keyword ooni           \
    -main-command 'curl -sm5 --connect-to ::example.com: https://ooni.io'
}

function main() {
  runtest http_got_nothing
  runtest http_recv_error
  runtest http_operation_timedout
  runtest http_couldnt_connect
  runtest http_blockpage
  runtest dns_injection
  runtest dns_timeout
  runtest dns_nxdomain
  runtest sni_man_in_the_middle
  runtest sni_got_nothing
  runtest sni_connect_error
}

main "$@"
