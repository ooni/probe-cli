#!/usr/bin/env python3


""" ./QA/probeasn.py - QA script for the -g miniooni option. """

import contextlib
import json
import os
import shlex
import shutil
import socket
import subprocess
import sys
import time
import urllib.parse

sys.path.insert(0, ".")
import common


def execute_miniooni(ooni_exe, outfile, arguments):
    """ Executes miniooni and returns the whole measurement. """
    if "miniooni" not in ooni_exe:
        return None
    tmpoutfile = "/tmp/{}".format(outfile)
    with contextlib.suppress(FileNotFoundError):
        os.remove(tmpoutfile)  # just in case
    cmdline = [
        ooni_exe,
        arguments,
        "-o",
        tmpoutfile,
        "--home",
        "/tmp",
        "example",
    ]
    print("exec: {}".format(cmdline))
    common.execute(cmdline)
    shutil.copy(tmpoutfile, outfile)
    result = common.read_result(outfile)
    assert isinstance(result, dict)
    assert isinstance(result["test_keys"], dict)
    return result


def probeasn_without_g_option(ooni_exe, outfile):
    """ Test case where we're not passing to miniooni the -g option """
    m = execute_miniooni(ooni_exe, outfile, "-n")
    if m is None:
        return
    assert m["probe_cc"] != "ZZ"
    assert m["probe_ip"] == "127.0.0.1"
    assert m["probe_asn"] != "AS0"
    assert m["probe_network_name"] != ""
    assert m["resolver_ip"] == "127.0.0.2"
    assert m["resolver_asn"] != "AS0"
    assert m["resolver_network_name"] != ""


def probeasn_with_g_option(ooni_exe, outfile):
    """ Test case where we're passing the -g option """
    m = execute_miniooni(ooni_exe, outfile, "-gn")
    if m is None:
        return
    assert m["probe_cc"] != "ZZ"
    assert m["probe_ip"] == "127.0.0.1"
    assert m["probe_asn"] == "AS0"
    assert m["probe_network_name"] == ""
    assert m["resolver_ip"] == "127.0.0.2"
    assert m["resolver_asn"] == "AS0"
    assert m["resolver_network_name"] == ""


def main():
    if len(sys.argv) != 2:
        sys.exit("usage: %s /path/to/ooniprobelegacy-like/binary" % sys.argv[0])
    outfile = "probeasn.jsonl"
    ooni_exe = sys.argv[1]
    tests = [
        probeasn_with_g_option,
        probeasn_without_g_option,
    ]
    for test in tests:
        test(ooni_exe, outfile)


if __name__ == "__main__":
    main()
