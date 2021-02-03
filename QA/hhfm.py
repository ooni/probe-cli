#!/usr/bin/env python3


""" ./QA/hhfm.py - main QA script for hhfm

    This script performs a bunch of hhfm tests under censored
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
        ooni_exe, outfile, "http_header_field_manipulation", tag, args
    )
    # TODO(bassosimone): what checks to put here?
    return tk


def hhfm_transparent_proxy(ooni_exe, outfile):
    """ Test case where we're passing through a transparent proxy """
    args = ["-iptables-hijack-http-to", "127.0.0.1:80"]
    tk = execute_jafar_and_return_validated_test_keys(
        ooni_exe, outfile, "hhfm_transparent_proxy", args,
    )
    # The proxy sees a domain that does not make any sense and does not
    # otherwise know where to connect to. Hence the most likely result is
    # a `dns_nxdomain_error` with total tampering.
    assert tk["tampering"]["header_field_name"] == False
    assert tk["tampering"]["header_field_number"] == False
    assert tk["tampering"]["header_field_value"] == False
    assert tk["tampering"]["header_name_capitalization"] == False
    assert tk["tampering"]["header_name_diff"] == []
    assert tk["tampering"]["request_line_capitalization"] == False
    assert tk["tampering"]["total"] == True


def main():
    if len(sys.argv) != 2:
        sys.exit("usage: %s /path/to/ooniprobelegacy-like/binary" % sys.argv[0])
    outfile = "hhfm.jsonl"
    ooni_exe = sys.argv[1]
    tests = [
        hhfm_transparent_proxy,
    ]
    for test in tests:
        test(ooni_exe, outfile)
        time.sleep(7)


if __name__ == "__main__":
    main()
