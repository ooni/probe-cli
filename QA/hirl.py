#!/usr/bin/env python3


""" ./QA/hirl.py - main QA script for hirl

    This script performs a bunch of hirl tests under censored
    network conditions and verifies that the measurement is consistent
    with the expectations, by parsing the resulting JSONL. """

import contextlib
import json
import os
import shlex
import socket
import subprocess
import sys
import time
import urllib.parse

sys.path.insert(0, ".")
import common


def execute_jafar_and_return_validated_test_keys(ooni_exe, outfile, tag, args):
    """ Executes jafar and returns the validated parsed test keys, or throws
        an AssertionError if the result is not valid. """
    tk = common.execute_jafar_and_miniooni(
        ooni_exe, outfile, "http_invalid_request_line", tag, args
    )
    # TODO(bassosimone): what checks to put here?
    return tk


def hirl_transparent_proxy(ooni_exe, outfile):
    """ Test case where we're passing through a transparent proxy """
    args = ["-iptables-hijack-http-to", "127.0.0.1:80"]
    tk = execute_jafar_and_return_validated_test_keys(
        ooni_exe, outfile, "hirl_transparent_proxy", args,
    )
    count = 0
    for entry in tk["failure_list"]:
        if entry is None:
            count += 1
        elif entry == "eof_error":
            count += 1e03
        else:
            count += 1e06
    assert count == 3002
    assert tk["tampering_list"] == [True, True, True, True, True]
    assert tk["tampering"] == True


def main():
    if len(sys.argv) != 2:
        sys.exit("usage: %s /path/to/ooniprobelegacy-like/binary" % sys.argv[0])
    outfile = "hirl.jsonl"
    ooni_exe = sys.argv[1]
    tests = [
        hirl_transparent_proxy,
    ]
    for test in tests:
        test(ooni_exe, outfile)
        time.sleep(7)


if __name__ == "__main__":
    main()
