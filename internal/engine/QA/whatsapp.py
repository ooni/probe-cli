#!/usr/bin/env python3


""" ./QA/whatsapp.py - main QA script for whatsapp

    This script performs a bunch of whatsapp tests under censored
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
    tk = common.execute_jafar_and_miniooni(ooni_exe, outfile, "whatsapp", tag, args)
    assert isinstance(tk["requests"], list)
    assert len(tk["requests"]) > 0
    for entry in tk["requests"]:
        assert isinstance(entry, dict)
        failure = entry["failure"]
        assert isinstance(failure, str) or failure is None
        assert isinstance(entry["request"], dict)
        req = entry["request"]
        common.check_maybe_binary_value(req["body"])
        assert isinstance(req["headers"], dict)
        for key, value in req["headers"].items():
            assert isinstance(key, str)
            common.check_maybe_binary_value(value)
        assert isinstance(req["method"], str)
        assert isinstance(entry["response"], dict)
        resp = entry["response"]
        common.check_maybe_binary_value(resp["body"])
        assert isinstance(resp["code"], int)
        if resp["headers"] is not None:
            for key, value in resp["headers"].items():
                assert isinstance(key, str)
                common.check_maybe_binary_value(value)
    assert isinstance(tk["tcp_connect"], list)
    assert len(tk["tcp_connect"]) > 0
    for entry in tk["tcp_connect"]:
        assert isinstance(entry, dict)
        assert isinstance(entry["ip"], str)
        assert isinstance(entry["port"], int)
        assert isinstance(entry["status"], dict)
        failure = entry["status"]["failure"]
        success = entry["status"]["success"]
        assert isinstance(failure, str) or failure is None
        assert isinstance(success, bool)
    return tk


def helper_for_blocking_endpoints(start, stop):
    """ Helper function for generating args for blocking endpoints """
    args = []
    for num in range(start, stop):
        args.append("-iptables-reset-ip")
        args.append("e{}.whatsapp.net".format(num))
    return args


def args_for_blocking_all_endpoints():
    """ Returns the arguments useful for blocking all endpoints """
    return helper_for_blocking_endpoints(1, 17)


def args_for_blocking_some_endpoints():
    """ Returns the arguments useful for blocking some endpoints """
    # Implementation note: apparently all the endpoints are now using just
    # four IP addresses, hence here we block some endpoints via DNS.
    #
    # TODO(bassosimone): this fact calls for creating an issue for making
    # the whatsapp experiment implementation more efficient.
    args = []
    args.append("-iptables-hijack-dns-to")
    args.append("127.0.0.1:53")
    for n in range(1, 7):
        args.append("-dns-proxy-block")
        args.append("e{}.whatsapp.net".format(n))
    return args


def args_for_blocking_v_whatsapp_net_https():
    """ Returns arguments for blocking v.whatsapp.net over https """
    #
    #  00 00          <SNI extension ID>
    #  00 13          <full extension length>
    #  00 11          <first entry length>
    #  00             <DNS hostname type>
    #  00 0e          <string length>
    #  76 2e ... 74   v.whatsapp.net
    #
    return [
        "-iptables-reset-keyword-hex",
        "|00 00 00 13 00 11 00 00 0e 76 2e 77 68 61 74 73 61 70 70 2e 6e 65 74|",
    ]


def args_for_blocking_web_whatsapp_com_http():
    """ Returns arguments for blocking web.whatsapp.com over http """
    return ["-iptables-reset-keyword", "Host: web.whatsapp.com"]


def args_for_blocking_web_whatsapp_com_https():
    """ Returns arguments for blocking web.whatsapp.com over https """
    #
    #  00 00          <SNI extension ID>
    #  00 15          <full extension length>
    #  00 13          <first entry length>
    #  00             <DNS hostname type>
    #  00 10          <string length>
    #  77 65 ... 6d   web.whatsapp.com
    #
    return [
        "-iptables-reset-keyword-hex",
        "|00 00 00 15 00 13 00 00 10 77 65 62 2e 77 68 61 74 73 61 70 70 2e 63 6f 6d|",
    ]


def whatsapp_block_everything(ooni_exe, outfile):
    """ Test case where everything we measure is blocked """
    args = []
    args.extend(args_for_blocking_all_endpoints())
    args.extend(args_for_blocking_v_whatsapp_net_https())
    args.extend(args_for_blocking_web_whatsapp_com_https())
    args.extend(args_for_blocking_web_whatsapp_com_http())
    tk = execute_jafar_and_return_validated_test_keys(
        ooni_exe, outfile, "whatsapp_block_everything", args,
    )
    assert tk["registration_server_failure"] == "connection_reset"
    assert tk["registration_server_status"] == "blocked"
    assert tk["whatsapp_endpoints_status"] == "blocked"
    assert tk["whatsapp_web_failure"] == "connection_reset"
    assert tk["whatsapp_web_status"] == "blocked"


def whatsapp_block_all_endpoints(ooni_exe, outfile):
    """ Test case where we only block whatsapp endpoints """
    args = args_for_blocking_all_endpoints()
    tk = execute_jafar_and_return_validated_test_keys(
        ooni_exe, outfile, "whatsapp_block_all_endpoints", args
    )
    assert tk["registration_server_failure"] == None
    assert tk["registration_server_status"] == "ok"
    assert tk["whatsapp_endpoints_status"] == "blocked"
    assert tk["whatsapp_web_failure"] == None
    assert tk["whatsapp_web_status"] == "ok"


def whatsapp_block_some_endpoints(ooni_exe, outfile):
    """ Test case where we block some whatsapp endpoints """
    args = args_for_blocking_some_endpoints()
    tk = execute_jafar_and_return_validated_test_keys(
        ooni_exe, outfile, "whatsapp_block_some_endpoints", args
    )
    assert tk["registration_server_failure"] == None
    assert tk["registration_server_status"] == "ok"
    assert tk["whatsapp_endpoints_status"] == "ok"
    assert tk["whatsapp_web_failure"] == None
    assert tk["whatsapp_web_status"] == "ok"


def whatsapp_block_registration_server(ooni_exe, outfile):
    """ Test case where we block the registration server """
    args = []
    args.extend(args_for_blocking_v_whatsapp_net_https())
    tk = execute_jafar_and_return_validated_test_keys(
        ooni_exe, outfile, "whatsapp_block_registration_server", args,
    )
    assert tk["registration_server_failure"] == "connection_reset"
    assert tk["registration_server_status"] == "blocked"
    assert tk["whatsapp_endpoints_status"] == "ok"
    assert tk["whatsapp_web_failure"] == None
    assert tk["whatsapp_web_status"] == "ok"


def whatsapp_block_web_http(ooni_exe, outfile):
    """ Test case where we block the HTTP web chat """
    args = []
    args.extend(args_for_blocking_web_whatsapp_com_http())
    tk = execute_jafar_and_return_validated_test_keys(
        ooni_exe, outfile, "whatsapp_block_web_http", args,
    )
    assert tk["registration_server_failure"] == None
    assert tk["registration_server_status"] == "ok"
    assert tk["whatsapp_endpoints_status"] == "ok"
    assert tk["whatsapp_web_failure"] == "connection_reset"
    assert tk["whatsapp_web_status"] == "blocked"


def whatsapp_block_web_https(ooni_exe, outfile):
    """ Test case where we block the HTTPS web chat """
    args = []
    args.extend(args_for_blocking_web_whatsapp_com_https())
    tk = execute_jafar_and_return_validated_test_keys(
        ooni_exe, outfile, "whatsapp_block_web_https", args,
    )
    assert tk["registration_server_failure"] == None
    assert tk["registration_server_status"] == "ok"
    assert tk["whatsapp_endpoints_status"] == "ok"
    assert tk["whatsapp_web_failure"] == "connection_reset"
    assert tk["whatsapp_web_status"] == "blocked"


def main():
    if len(sys.argv) != 2:
        sys.exit("usage: %s /path/to/ooniprobelegacy-like/binary" % sys.argv[0])
    outfile = "whatsapp.jsonl"
    ooni_exe = sys.argv[1]
    tests = [
        whatsapp_block_everything,
        whatsapp_block_all_endpoints,
        whatsapp_block_some_endpoints,
        whatsapp_block_registration_server,
        whatsapp_block_web_http,
        whatsapp_block_web_https,
    ]
    for test in tests:
        test(ooni_exe, outfile)
        time.sleep(7)


if __name__ == "__main__":
    main()
