#!/usr/bin/env python3


""" ./QA/fbmessenger.py - main QA script for fbmessenger

    This script performs a bunch of fbmessenger tests under censored
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


services = {
    "stun": "stun.fbsbx.com",
    "b_api": "b-api.facebook.com",
    "b_graph": "b-graph.facebook.com",
    "edge": "edge-mqtt.facebook.com",
    "external_cdn": "external.xx.fbcdn.net",
    "scontent_cdn": "scontent.xx.fbcdn.net",
    "star": "star.c10r.facebook.com",
}


def execute_jafar_and_return_validated_test_keys(ooni_exe, outfile, tag, args):
    """ Executes jafar and returns the validated parsed test keys, or throws
        an AssertionError if the result is not valid. """
    tk = common.execute_jafar_and_miniooni(
        ooni_exe, outfile, "facebook_messenger", tag, args
    )
    assert tk["requests"] is None
    if tk["tcp_connect"] is not None:
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


def helper_for_blocking_services_via_dns(service):
    """ Helper for hijacking a service via dns """
    args = []
    args.append("-iptables-hijack-dns-to")
    args.append("127.0.0.1:53")
    args.append("-dns-proxy-block")
    args.append(service)
    return args


def helper_for_hijacking_services_via_dns(service):
    """ Helper for hijacking a service via dns """
    args = []
    args.append("-iptables-hijack-dns-to")
    args.append("127.0.0.1:53")
    args.append("-dns-proxy-hijack")
    args.append(service)
    return args


def helper_for_blocking_services_via_tcp(service):
    """ Helper for blocking a service via tcp """
    args = []
    args.append("-iptables-reset-ip")
    args.append(service)
    return args


def fbmessenger_dns_hijacked_for_all(ooni_exe, outfile):
    """ Test case where everything we measure is DNS hijacked """
    args = []
    for _, value in services.items():
        args.extend(helper_for_hijacking_services_via_dns(value))
    tk = execute_jafar_and_return_validated_test_keys(
        ooni_exe, outfile, "fbmessenger_dns_hijacked_for_all", args,
    )
    assert tk["facebook_b_api_dns_consistent"] == False
    assert tk["facebook_b_api_reachable"] == None
    assert tk["facebook_b_graph_dns_consistent"] == False
    assert tk["facebook_b_graph_reachable"] == None
    assert tk["facebook_edge_dns_consistent"] == False
    assert tk["facebook_edge_reachable"] == None
    assert tk["facebook_external_cdn_dns_consistent"] == False
    assert tk["facebook_external_cdn_reachable"] == None
    assert tk["facebook_scontent_cdn_dns_consistent"] == False
    assert tk["facebook_scontent_cdn_reachable"] == None
    assert tk["facebook_star_dns_consistent"] == False
    assert tk["facebook_star_reachable"] == None
    assert tk["facebook_stun_dns_consistent"] == False
    assert tk["facebook_stun_reachable"] == None
    assert tk["facebook_dns_blocking"] == True
    assert tk["facebook_tcp_blocking"] == False


def fbmessenger_dns_hijacked_for_some(ooni_exe, outfile):
    """ Test case where some endpoints are DNS hijacked """
    args = []
    args.extend(helper_for_hijacking_services_via_dns(services["star"]))
    args.extend(helper_for_hijacking_services_via_dns(services["edge"]))
    tk = execute_jafar_and_return_validated_test_keys(
        ooni_exe, outfile, "fbmessenger_dns_hijacked_for_some", args,
    )
    assert tk["facebook_b_api_dns_consistent"] == True
    assert tk["facebook_b_api_reachable"] == True
    assert tk["facebook_b_graph_dns_consistent"] == True
    assert tk["facebook_b_graph_reachable"] == True
    assert tk["facebook_edge_dns_consistent"] == False
    assert tk["facebook_edge_reachable"] == None
    assert tk["facebook_external_cdn_dns_consistent"] == True
    assert tk["facebook_external_cdn_reachable"] == True
    assert tk["facebook_scontent_cdn_dns_consistent"] == True
    assert tk["facebook_scontent_cdn_reachable"] == True
    assert tk["facebook_star_dns_consistent"] == False
    assert tk["facebook_star_reachable"] == None
    assert tk["facebook_stun_dns_consistent"] == True
    assert tk["facebook_stun_reachable"] == None
    assert tk["facebook_dns_blocking"] == True
    assert tk["facebook_tcp_blocking"] == False


def fbmessenger_dns_blocked_for_all(ooni_exe, outfile):
    """ Test case where everything we measure is DNS blocked """
    args = []
    for _, value in services.items():
        args.extend(helper_for_blocking_services_via_dns(value))
    tk = execute_jafar_and_return_validated_test_keys(
        ooni_exe, outfile, "fbmessenger_dns_blocked_for_all", args,
    )
    assert tk["facebook_b_api_dns_consistent"] == False
    assert tk["facebook_b_api_reachable"] == None
    assert tk["facebook_b_graph_dns_consistent"] == False
    assert tk["facebook_b_graph_reachable"] == None
    assert tk["facebook_edge_dns_consistent"] == False
    assert tk["facebook_edge_reachable"] == None
    assert tk["facebook_external_cdn_dns_consistent"] == False
    assert tk["facebook_external_cdn_reachable"] == None
    assert tk["facebook_scontent_cdn_dns_consistent"] == False
    assert tk["facebook_scontent_cdn_reachable"] == None
    assert tk["facebook_star_dns_consistent"] == False
    assert tk["facebook_star_reachable"] == None
    assert tk["facebook_stun_dns_consistent"] == False
    assert tk["facebook_stun_reachable"] == None
    assert tk["facebook_dns_blocking"] == True
    assert tk["facebook_tcp_blocking"] == False


def fbmessenger_dns_blocked_for_some(ooni_exe, outfile):
    """ Test case where some endpoints are DNS blocked """
    args = []
    args.extend(helper_for_blocking_services_via_dns(services["b_graph"]))
    args.extend(helper_for_blocking_services_via_dns(services["stun"]))
    tk = execute_jafar_and_return_validated_test_keys(
        ooni_exe, outfile, "fbmessenger_dns_blocked_for_some", args,
    )
    assert tk["facebook_b_api_dns_consistent"] == True
    assert tk["facebook_b_api_reachable"] == True
    assert tk["facebook_b_graph_dns_consistent"] == False
    assert tk["facebook_b_graph_reachable"] == None
    assert tk["facebook_edge_dns_consistent"] == True
    assert tk["facebook_edge_reachable"] == True
    assert tk["facebook_external_cdn_dns_consistent"] == True
    assert tk["facebook_external_cdn_reachable"] == True
    assert tk["facebook_scontent_cdn_dns_consistent"] == True
    assert tk["facebook_scontent_cdn_reachable"] == True
    assert tk["facebook_star_dns_consistent"] == True
    assert tk["facebook_star_reachable"] == True
    assert tk["facebook_stun_dns_consistent"] == False
    assert tk["facebook_stun_reachable"] == None
    assert tk["facebook_dns_blocking"] == True
    assert tk["facebook_tcp_blocking"] == False


def fbmessenger_tcp_blocked_for_all(ooni_exe, outfile):
    """ Test case where everything we measure is TCP blocked """
    args = []
    for _, value in services.items():
        args.extend(helper_for_blocking_services_via_tcp(value))
    tk = execute_jafar_and_return_validated_test_keys(
        ooni_exe, outfile, "fbmessenger_tcp_blocked_for_all", args,
    )
    assert tk["facebook_b_api_dns_consistent"] == True
    assert tk["facebook_b_api_reachable"] == False
    assert tk["facebook_b_graph_dns_consistent"] == True
    assert tk["facebook_b_graph_reachable"] == False
    assert tk["facebook_edge_dns_consistent"] == True
    assert tk["facebook_edge_reachable"] == False
    assert tk["facebook_external_cdn_dns_consistent"] == True
    assert tk["facebook_external_cdn_reachable"] == False
    assert tk["facebook_scontent_cdn_dns_consistent"] == True
    assert tk["facebook_scontent_cdn_reachable"] == False
    assert tk["facebook_star_dns_consistent"] == True
    assert tk["facebook_star_reachable"] == False
    assert tk["facebook_stun_dns_consistent"] == True
    assert tk["facebook_stun_reachable"] == None
    assert tk["facebook_dns_blocking"] == False
    assert tk["facebook_tcp_blocking"] == True


def fbmessenger_tcp_blocked_for_some(ooni_exe, outfile):
    """ Test case where only some endpoints are TCP blocked """
    args = []
    args.extend(helper_for_blocking_services_via_tcp(services["edge"]))
    tk = execute_jafar_and_return_validated_test_keys(
        ooni_exe, outfile, "fbmessenger_tcp_blocked_for_some", args,
    )
    assert tk["facebook_b_api_dns_consistent"] == True
    assert tk["facebook_b_api_reachable"] == True
    assert tk["facebook_b_graph_dns_consistent"] == True
    assert tk["facebook_b_graph_reachable"] == True
    assert tk["facebook_edge_dns_consistent"] == True
    assert tk["facebook_edge_reachable"] == False
    assert tk["facebook_external_cdn_dns_consistent"] == True
    assert tk["facebook_external_cdn_reachable"] == True
    assert tk["facebook_scontent_cdn_dns_consistent"] == True
    assert tk["facebook_scontent_cdn_reachable"] == True
    assert tk["facebook_star_dns_consistent"] == True
    assert tk["facebook_star_reachable"] == True
    assert tk["facebook_stun_dns_consistent"] == True
    assert tk["facebook_stun_reachable"] == None
    assert tk["facebook_dns_blocking"] == False
    assert tk["facebook_tcp_blocking"] == True


def fbmessenger_mixed_results(ooni_exe, outfile):
    """ Test case where only some endpoints are TCP blocked """
    args = []
    args.extend(helper_for_blocking_services_via_tcp(services["edge"]))
    args.extend(helper_for_blocking_services_via_dns(services["b_api"]))
    tk = execute_jafar_and_return_validated_test_keys(
        ooni_exe, outfile, "fbmessenger_tcp_blocked_for_some", args,
    )
    assert tk["facebook_b_api_dns_consistent"] == False
    assert tk["facebook_b_api_reachable"] == None
    assert tk["facebook_b_graph_dns_consistent"] == True
    assert tk["facebook_b_graph_reachable"] == True
    assert tk["facebook_edge_dns_consistent"] == True
    assert tk["facebook_edge_reachable"] == False
    assert tk["facebook_external_cdn_dns_consistent"] == True
    assert tk["facebook_external_cdn_reachable"] == True
    assert tk["facebook_scontent_cdn_dns_consistent"] == True
    assert tk["facebook_scontent_cdn_reachable"] == True
    assert tk["facebook_star_dns_consistent"] == True
    assert tk["facebook_star_reachable"] == True
    assert tk["facebook_stun_dns_consistent"] == True
    assert tk["facebook_stun_reachable"] == None
    assert tk["facebook_dns_blocking"] == True
    assert tk["facebook_tcp_blocking"] == True


def main():
    if len(sys.argv) != 2:
        sys.exit("usage: %s /path/to/ooniprobelegacy-like/binary" % sys.argv[0])
    outfile = "fbmessenger.jsonl"
    ooni_exe = sys.argv[1]
    tests = [
        fbmessenger_dns_hijacked_for_all,
        fbmessenger_dns_hijacked_for_some,
        fbmessenger_dns_blocked_for_all,
        fbmessenger_dns_blocked_for_some,
        fbmessenger_tcp_blocked_for_all,
        fbmessenger_tcp_blocked_for_some,
        fbmessenger_mixed_results,
    ]
    for test in tests:
        test(ooni_exe, outfile)
        time.sleep(7)


if __name__ == "__main__":
    main()
