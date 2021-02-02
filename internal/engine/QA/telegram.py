#!/usr/bin/env python3


""" ./QA/telegram.py - main QA script for telegram

    This script performs a bunch of telegram tests under censored
    network conditions and verifies that the measurement is consistent
    with the expectations, by parsing the resulting JSONL. """

import contextlib
import json
import os
import shlex
import subprocess
import sys
import time
import urllib.parse

sys.path.insert(0, ".")
import common


ALL_POP_IPS = (
    "149.154.175.50",
    "149.154.167.51",
    "149.154.175.100",
    "149.154.167.91",
    "149.154.171.5",
)


def execute_jafar_and_return_validated_test_keys(ooni_exe, outfile, tag, args):
    """ Executes jafar and returns the validated parsed test keys, or throws
        an AssertionError if the result is not valid. """
    tk = common.execute_jafar_and_miniooni(ooni_exe, outfile, "telegram", tag, args)
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


def args_for_blocking_all_pop_ips():
    """ Returns the arguments useful for blocking all POPs IPs """
    args = []
    for ip in ALL_POP_IPS:
        args.append("-iptables-reset-ip")
        args.append(ip)
    return args


def args_for_blocking_web_telegram_org_http():
    """ Returns arguments for blocking web.telegram.org over http """
    return ["-iptables-reset-keyword", "Host: web.telegram.org"]


def args_for_blocking_web_telegram_org_https():
    """ Returns arguments for blocking web.telegram.org over https """
    #
    #  00 00          <SNI extension ID>
    #  00 15          <full extension length>
    #  00 13          <first entry length>
    #  00             <DNS hostname type>
    #  00 10          <string length>
    #  77 65 ... 67   web.telegram.org
    #
    return [
        "-iptables-reset-keyword-hex",
        "|00 00 00 15 00 13 00 00 10 77 65 62 2e 74 65 6c 65 67 72 61 6d 2e 6f 72 67|",
    ]


def telegram_block_everything(ooni_exe, outfile):
    """ Test case where everything we measure is blocked """
    args = []
    args.extend(args_for_blocking_all_pop_ips())
    args.extend(args_for_blocking_web_telegram_org_https())
    args.extend(args_for_blocking_web_telegram_org_http())
    tk = execute_jafar_and_return_validated_test_keys(
        ooni_exe, outfile, "telegram_block_everything", args,
    )
    assert tk["telegram_tcp_blocking"] == True
    assert tk["telegram_http_blocking"] == True
    assert tk["telegram_web_failure"] == "connection_reset"
    assert tk["telegram_web_status"] == "blocked"


def telegram_tcp_blocking_all(ooni_exe, outfile):
    """ Test case where all POPs are TCP/IP blocked """
    args = args_for_blocking_all_pop_ips()
    tk = execute_jafar_and_return_validated_test_keys(
        ooni_exe, outfile, "telegram_tcp_blocking_all", args
    )
    assert tk["telegram_tcp_blocking"] == True
    assert tk["telegram_http_blocking"] == True
    assert tk["telegram_web_failure"] == None
    assert tk["telegram_web_status"] == "ok"


def telegram_tcp_blocking_some(ooni_exe, outfile):
    """ Test case where some POPs are TCP/IP blocked """
    args = [
        "-iptables-reset-ip",
        ALL_POP_IPS[0],
    ]
    tk = execute_jafar_and_return_validated_test_keys(
        ooni_exe, outfile, "telegram_tcp_blocking_some", args
    )
    assert tk["telegram_tcp_blocking"] == False
    assert tk["telegram_http_blocking"] == False
    assert tk["telegram_web_failure"] == None
    assert tk["telegram_web_status"] == "ok"


def telegram_http_blocking_all(ooni_exe, outfile):
    """ Test case where all POPs are HTTP blocked """
    args = []
    for ip in ALL_POP_IPS:
        args.append("-iptables-reset-keyword")
        args.append(ip)
    tk = execute_jafar_and_return_validated_test_keys(
        ooni_exe, outfile, "telegram_http_blocking_all", args,
    )
    assert tk["telegram_tcp_blocking"] == False
    assert tk["telegram_http_blocking"] == True
    assert tk["telegram_web_failure"] == None
    assert tk["telegram_web_status"] == "ok"


def telegram_http_blocking_some(ooni_exe, outfile):
    """ Test case where some POPs are HTTP blocked """
    args = [
        "-iptables-reset-keyword",
        ALL_POP_IPS[0],
    ]
    tk = execute_jafar_and_return_validated_test_keys(
        ooni_exe, outfile, "telegram_http_blocking_some", args,
    )
    assert tk["telegram_tcp_blocking"] == False
    assert tk["telegram_http_blocking"] == False
    assert tk["telegram_web_failure"] == None
    assert tk["telegram_web_status"] == "ok"


def telegram_web_failure_http(ooni_exe, outfile):
    """ Test case where the web HTTP endpoint is blocked """
    args = args_for_blocking_web_telegram_org_http()
    tk = execute_jafar_and_return_validated_test_keys(
        ooni_exe, outfile, "telegram_web_failure_http", args,
    )
    assert tk["telegram_tcp_blocking"] == False
    assert tk["telegram_http_blocking"] == False
    assert tk["telegram_web_failure"] == "connection_reset"
    assert tk["telegram_web_status"] == "blocked"


def telegram_web_failure_https(ooni_exe, outfile):
    """ Test case where the web HTTPS endpoint is blocked """
    args = args_for_blocking_web_telegram_org_https()
    tk = execute_jafar_and_return_validated_test_keys(
        ooni_exe, outfile, "telegram_web_failure_https", args,
    )
    assert tk["telegram_tcp_blocking"] == False
    assert tk["telegram_http_blocking"] == False
    assert tk["telegram_web_failure"] == "connection_reset"
    assert tk["telegram_web_status"] == "blocked"


def main():
    if len(sys.argv) != 2:
        sys.exit("usage: %s /path/to/ooniprobelegacy-like/binary" % sys.argv[0])
    outfile = "telegram.jsonl"
    ooni_exe = sys.argv[1]
    tests = [
        telegram_block_everything,
        telegram_tcp_blocking_all,
        telegram_tcp_blocking_some,
        telegram_http_blocking_all,
        telegram_http_blocking_some,
        telegram_web_failure_http,
        telegram_web_failure_https,
    ]
    for test in tests:
        test(ooni_exe, outfile)
        time.sleep(7)


if __name__ == "__main__":
    main()
