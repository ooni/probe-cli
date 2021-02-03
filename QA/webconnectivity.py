#!/usr/bin/env python3


""" ./QA/webconnectivity.py - main QA script for webconnectivity

    This script performs a bunch of webconnectivity tests under censored
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


def execute_jafar_and_return_validated_test_keys(
    ooni_exe, outfile, experiment_args, tag, args
):
    """ Executes jafar and returns the validated parsed test keys, or throws
        an AssertionError if the result is not valid. """
    tk = common.execute_jafar_and_miniooni(
        ooni_exe, outfile, experiment_args, tag, args
    )
    return tk


def assert_status_flags_are(ooni_exe, tk, desired):
    """ Checks whether the status flags are what we expect them to
        be when we're running miniooni. This check only makes sense
        with miniooni b/c status flags are a miniooni extension. """
    if "miniooni" not in ooni_exe:
        return
    assert tk["x_status"] == desired


def webconnectivity_https_ok_with_control_failure(ooni_exe, outfile):
    """ Successful HTTPS measurement but control failure. """
    args = [
        "-iptables-reset-keyword",
        "wcth.ooni.io",
    ]
    tk = execute_jafar_and_return_validated_test_keys(
        ooni_exe,
        outfile,
        "-i https://example.com/ web_connectivity",
        "webconnectivity_https_ok_with_control_failure",
        args,
    )
    assert tk["dns_experiment_failure"] == None
    assert tk["dns_consistency"] == None
    assert tk["control_failure"] == "connection_reset"
    assert tk["http_experiment_failure"] == None
    assert tk["body_length_match"] == None
    assert tk["body_proportion"] == 0
    assert tk["status_code_match"] == None
    assert tk["headers_match"] == None
    assert tk["title_match"] == None
    if "miniooni" in ooni_exe:
        assert tk["blocking"] == False
        assert tk["accessible"] == True
    else:
        assert tk["blocking"] == None
        assert tk["accessible"] == None
    assert_status_flags_are(ooni_exe, tk, 1)


def webconnectivity_http_ok_with_control_failure(ooni_exe, outfile):
    """ Successful HTTP measurement but control failure. """
    args = [
        "-iptables-reset-keyword",
        "wcth.ooni.io",
    ]
    tk = execute_jafar_and_return_validated_test_keys(
        ooni_exe,
        outfile,
        "-i http://example.org/ web_connectivity",
        "webconnectivity_http_ok_with_control_failure",
        args,
    )
    assert tk["dns_experiment_failure"] == None
    assert tk["dns_consistency"] == None
    assert tk["control_failure"] == "connection_reset"
    assert tk["http_experiment_failure"] == None
    assert tk["body_length_match"] == None
    assert tk["body_proportion"] == 0
    assert tk["status_code_match"] == None
    assert tk["headers_match"] == None
    assert tk["title_match"] == None
    assert tk["blocking"] == None
    assert tk["accessible"] == None
    assert_status_flags_are(ooni_exe, tk, 8)


def webconnectivity_transparent_http_proxy(ooni_exe, outfile):
    """ Test case where we pass through a transparent HTTP proxy """
    args = []
    args.append("-iptables-hijack-https-to")
    args.append("127.0.0.1:443")
    tk = execute_jafar_and_return_validated_test_keys(
        ooni_exe,
        outfile,
        "-i https://example.org web_connectivity",
        "webconnectivity_transparent_http_proxy",
        args,
    )
    assert tk["dns_experiment_failure"] == None
    assert tk["dns_consistency"] == "consistent"
    assert tk["control_failure"] == None
    assert tk["http_experiment_failure"] == None
    assert tk["body_length_match"] == True
    assert tk["body_proportion"] == 1
    assert tk["status_code_match"] == True
    assert tk["headers_match"] == True
    assert tk["title_match"] == True
    assert tk["blocking"] == False
    assert tk["accessible"] == True
    assert_status_flags_are(ooni_exe, tk, 1)


def webconnectivity_dns_hijacking(ooni_exe, outfile):
    """ Test case where there is DNS hijacking towards a transparent proxy. """
    args = []
    args.append("-iptables-hijack-dns-to")
    args.append("127.0.0.1:53")
    args.append("-dns-proxy-hijack")
    args.append("example.org")
    tk = execute_jafar_and_return_validated_test_keys(
        ooni_exe,
        outfile,
        "-i https://example.org web_connectivity",
        "webconnectivity_dns_hijacking",
        args,
    )
    assert tk["dns_experiment_failure"] == None
    assert tk["dns_consistency"] == "inconsistent"
    assert tk["control_failure"] == None
    assert tk["http_experiment_failure"] == None
    assert tk["body_length_match"] == True
    assert tk["body_proportion"] == 1
    assert tk["status_code_match"] == True
    assert tk["headers_match"] == True
    assert tk["title_match"] == True
    assert tk["blocking"] == False
    assert tk["accessible"] == True
    assert_status_flags_are(ooni_exe, tk, 1)


def webconnectivity_control_unreachable_and_using_http(ooni_exe, outfile):
    """ Test case where the control is unreachable and we're using the
        plaintext HTTP protocol rather than HTTPS """
    args = []
    args.append("-iptables-reset-keyword")
    args.append("wcth.ooni.io")
    tk = execute_jafar_and_return_validated_test_keys(
        ooni_exe,
        outfile,
        "-i http://example.org web_connectivity",
        "webconnectivity_control_unreachable_and_using_http",
        args,
    )
    assert tk["dns_experiment_failure"] == None
    assert tk["dns_consistency"] == None
    assert tk["control_failure"] == "connection_reset"
    assert tk["http_experiment_failure"] == None
    assert tk["body_length_match"] == None
    assert tk["body_proportion"] == 0
    assert tk["status_code_match"] == None
    assert tk["headers_match"] == None
    assert tk["title_match"] == None
    assert tk["blocking"] == None
    assert tk["accessible"] == None
    assert_status_flags_are(ooni_exe, tk, 8)


def webconnectivity_nonexistent_domain(ooni_exe, outfile):
    """ Test case where the domain does not exist """
    args = []
    tk = execute_jafar_and_return_validated_test_keys(
        ooni_exe,
        outfile,
        "-i http://antani.ooni.io web_connectivity",
        "webconnectivity_nonexistent_domain",
        args,
    )
    # TODO(bassosimone): Debateable result. We need to do better here.
    # See <https://github.com/ooni/probe-engine/issues/579>.
    #
    # Note that MK is not doing it right here because it's suppressing the
    # dns_nxdomain_error that instead is very informative. Yet, it is reporting
    # a failure in HTTP, which miniooni does not because it does not make
    # sense to perform HTTP when there are no IP addresses.
    #
    # The following seems indeed a bug in MK where we don't properly record the
    # actual error that occurred when performing the DNS experiment.
    #
    # See <https://github.com/measurement-kit/measurement-kit/issues/1931>.
    if "miniooni" in ooni_exe:
        assert tk["dns_experiment_failure"] == "dns_nxdomain_error"
    else:
        assert tk["dns_experiment_failure"] == None
    assert tk["dns_consistency"] == "consistent"
    assert tk["control_failure"] == None
    if "miniooni" in ooni_exe:
        assert tk["http_experiment_failure"] == None
    else:
        assert tk["http_experiment_failure"] == "dns_lookup_error"
    assert tk["body_length_match"] == None
    assert tk["body_proportion"] == 0
    assert tk["status_code_match"] == None
    assert tk["headers_match"] == None
    assert tk["title_match"] == None
    assert tk["blocking"] == False
    assert tk["accessible"] == True
    assert_status_flags_are(ooni_exe, tk, 2052)


def webconnectivity_tcpip_blocking_with_consistent_dns(ooni_exe, outfile):
    """ Test case where there's TCP/IP blocking w/ consistent DNS """
    ip = socket.gethostbyname("nexa.polito.it")
    args = [
        "-iptables-drop-ip",
        ip,
    ]
    tk = execute_jafar_and_return_validated_test_keys(
        ooni_exe,
        outfile,
        "-i http://nexa.polito.it web_connectivity",
        "webconnectivity_tcpip_blocking_with_consistent_dns",
        args,
    )
    assert tk["dns_experiment_failure"] == None
    assert tk["dns_consistency"] == "consistent"
    assert tk["control_failure"] == None
    assert tk["http_experiment_failure"] == "generic_timeout_error"
    assert tk["body_length_match"] == None
    assert tk["body_proportion"] == 0
    assert tk["status_code_match"] == None
    assert tk["headers_match"] == None
    assert tk["title_match"] == None
    assert tk["blocking"] == "tcp_ip"
    assert tk["accessible"] == False
    assert_status_flags_are(ooni_exe, tk, 4224)


def webconnectivity_tcpip_blocking_with_inconsistent_dns(ooni_exe, outfile):
    """ Test case where there's TCP/IP blocking w/ inconsistent DNS """

    def runner(port):
        args = [
            "-dns-proxy-hijack",
            "nexa.polito.it",
            "-iptables-hijack-dns-to",
            "127.0.0.1:53",
            "-iptables-hijack-http-to",
            "127.0.0.1:{}".format(port),
        ]
        tk = execute_jafar_and_return_validated_test_keys(
            ooni_exe,
            outfile,
            "-i http://nexa.polito.it web_connectivity",
            "webconnectivity_tcpip_blocking_with_inconsistent_dns",
            args,
        )
        assert tk["dns_experiment_failure"] == None
        assert tk["dns_consistency"] == "inconsistent"
        assert tk["control_failure"] == None
        assert tk["http_experiment_failure"] == "connection_refused"
        assert tk["body_length_match"] == None
        assert tk["body_proportion"] == 0
        assert tk["status_code_match"] == None
        assert tk["headers_match"] == None
        assert tk["title_match"] == None
        assert tk["blocking"] == "dns"
        assert tk["accessible"] == False
        assert_status_flags_are(ooni_exe, tk, 4256)

    common.with_free_port(runner)


def webconnectivity_http_connection_refused_with_consistent_dns(ooni_exe, outfile):
    """ Test case where there's TCP/IP blocking w/ consistent DNS that occurs
        while we're following the chain of redirects. """
    # We use a bit.ly link redirecting to nexa.polito.it. We block the IP address
    # used by nexa.polito.it. So the error should happen in the redirect chain.
    ip = socket.gethostbyname("nexa.polito.it")
    args = [
        "-iptables-reset-ip",
        ip,
    ]
    tk = execute_jafar_and_return_validated_test_keys(
        ooni_exe,
        outfile,
        "-i https://bit.ly/3h9EJR3 web_connectivity",
        "webconnectivity_http_connection_refused_with_consistent_dns",
        args,
    )
    assert tk["dns_experiment_failure"] == None
    assert tk["dns_consistency"] == "consistent"
    assert tk["control_failure"] == None
    assert tk["http_experiment_failure"] == "connection_refused"
    assert tk["body_length_match"] == None
    assert tk["body_proportion"] == 0
    assert tk["status_code_match"] == None
    assert tk["headers_match"] == None
    assert tk["title_match"] == None
    assert tk["blocking"] == "http-failure"
    assert tk["accessible"] == False
    assert_status_flags_are(ooni_exe, tk, 8320)


def webconnectivity_http_connection_reset_with_consistent_dns(ooni_exe, outfile):
    """ Test case where there's RST-based blocking blocking w/ consistent DNS that
        occurs while we're following the chain of redirects. """
    # We use a bit.ly link redirecting to nexa.polito.it. We block the Host header
    # used for nexa.polito.it. So the error should happen in the redirect chain.
    args = [
        "-iptables-reset-keyword",
        "Host: nexa",
    ]
    tk = execute_jafar_and_return_validated_test_keys(
        ooni_exe,
        outfile,
        "-i https://bit.ly/3h9EJR3 web_connectivity",
        "webconnectivity_http_connection_reset_with_consistent_dns",
        args,
    )
    assert tk["dns_experiment_failure"] == None
    assert tk["dns_consistency"] == "consistent"
    assert tk["control_failure"] == None
    assert tk["http_experiment_failure"] == "connection_reset"
    assert tk["body_length_match"] == None
    assert tk["body_proportion"] == 0
    assert tk["status_code_match"] == None
    assert tk["headers_match"] == None
    assert tk["title_match"] == None
    assert tk["blocking"] == "http-failure"
    assert tk["accessible"] == False
    assert_status_flags_are(ooni_exe, tk, 8448)


def webconnectivity_http_nxdomain_with_consistent_dns(ooni_exe, outfile):
    """ Test case where there's a redirection and the redirected request cannot
        continue because a NXDOMAIN error occurs. """
    # We use a bit.ly link redirecting to nexa.polito.it. We block the DNS request
    # for nexa.polito.it. So the error should happen in the redirect chain.
    args = [
        "-iptables-hijack-dns-to",
        "127.0.0.1:53",
        "-dns-proxy-block",
        "nexa.polito.it",
    ]
    tk = execute_jafar_and_return_validated_test_keys(
        ooni_exe,
        outfile,
        "-i https://bit.ly/3h9EJR3 web_connectivity",
        "webconnectivity_http_nxdomain_with_consistent_dns",
        args,
    )
    assert tk["dns_experiment_failure"] == None
    assert tk["dns_consistency"] == "consistent"
    assert tk["control_failure"] == None
    assert (
        tk["http_experiment_failure"] == "dns_nxdomain_error"  # miniooni
        or tk["http_experiment_failure"] == "dns_lookup_error"  # MK
    )
    assert tk["body_length_match"] == None
    assert tk["body_proportion"] == 0
    assert tk["status_code_match"] == None
    assert tk["headers_match"] == None
    assert tk["title_match"] == None
    assert tk["blocking"] == "dns"
    assert tk["accessible"] == False
    assert_status_flags_are(ooni_exe, tk, 8224)


def webconnectivity_http_eof_error_with_consistent_dns(ooni_exe, outfile):
    """ Test case where there's a redirection and the redirected request cannot
        continue because an eof_error error occurs. """
    # We use a bit.ly link redirecting to nexa.polito.it. We block the HTTP request
    # for nexa.polito.it using the cleartext bad proxy. So the error should happen in
    # the redirect chain and should be EOF.
    args = [
        "-iptables-hijack-dns-to",
        "127.0.0.1:53",
        "-dns-proxy-hijack",
        "nexa.polito.it",
        "-iptables-hijack-http-to",
        "127.0.0.1:7117",  # this is badproxy's cleartext endpoint
    ]
    tk = execute_jafar_and_return_validated_test_keys(
        ooni_exe,
        outfile,
        "-i https://bit.ly/3h9EJR3 web_connectivity",  # bit.ly uses https
        "webconnectivity_http_eof_error_with_consistent_dns",
        args,
    )
    assert tk["dns_experiment_failure"] == None
    assert tk["dns_consistency"] == "consistent"
    assert tk["control_failure"] == None
    assert tk["http_experiment_failure"] == "eof_error"
    assert tk["body_length_match"] == None
    assert tk["body_proportion"] == 0
    assert tk["status_code_match"] == None
    assert tk["headers_match"] == None
    assert tk["title_match"] == None
    assert tk["blocking"] == "http-failure"
    assert tk["accessible"] == False
    assert_status_flags_are(ooni_exe, tk, 8448)


def webconnectivity_http_generic_timeout_error_with_consistent_dns(ooni_exe, outfile):
    """ Test case where there's a redirection and the redirected request cannot
        continue because a generic_timeout_error error occurs. """
    # We use a bit.ly link redirecting to nexa.polito.it. We block the HTTP request
    # for nexa.polito.it by dropping packets using DPI. So the error should happen in
    # the redirect chain and should be timeout.
    args = [
        "-iptables-hijack-dns-to",
        "127.0.0.1:53",
        "-dns-proxy-hijack",
        "nexa.polito.it",
        "-iptables-drop-keyword",
        "Host: nexa",
    ]
    tk = execute_jafar_and_return_validated_test_keys(
        ooni_exe,
        outfile,
        "-i https://bit.ly/3h9EJR3 web_connectivity",
        "webconnectivity_http_generic_timeout_error_with_consistent_dns",
        args,
    )
    assert tk["dns_experiment_failure"] == None
    assert tk["dns_consistency"] == "consistent"
    assert tk["control_failure"] == None
    assert tk["http_experiment_failure"] == "generic_timeout_error"
    assert tk["body_length_match"] == None
    assert tk["body_proportion"] == 0
    assert tk["status_code_match"] == None
    assert tk["headers_match"] == None
    assert tk["title_match"] == None
    assert tk["blocking"] == "http-failure"
    assert tk["accessible"] == False
    assert_status_flags_are(ooni_exe, tk, 8704)


def webconnectivity_http_connection_reset_with_inconsistent_dns(ooni_exe, outfile):
    """ Test case where there's inconsistent DNS and the connection is RST when
        we're executing HTTP code. """
    args = [
        "-iptables-reset-keyword",
        "nexa.polito.it",
        "-iptables-hijack-dns-to",
        "127.0.0.1:53",
        "-dns-proxy-hijack",
        "polito",
    ]
    tk = execute_jafar_and_return_validated_test_keys(
        ooni_exe,
        outfile,
        "-i http://nexa.polito.it/ web_connectivity",
        "webconnectivity_http_connection_reset_with_inconsistent_dns",
        args,
    )
    assert tk["dns_experiment_failure"] == None
    assert tk["dns_consistency"] == "inconsistent"
    assert tk["control_failure"] == None
    assert tk["http_experiment_failure"] == "connection_reset"
    assert tk["body_length_match"] == None
    assert tk["body_proportion"] == 0
    assert tk["status_code_match"] == None
    assert tk["headers_match"] == None
    assert tk["title_match"] == None
    assert tk["blocking"] == "dns"
    assert tk["accessible"] == False
    assert_status_flags_are(ooni_exe, tk, 8480)


def webconnectivity_http_successful_website(ooni_exe, outfile):
    """ Test case where we succeed with an HTTP only webpage """
    args = []
    tk = execute_jafar_and_return_validated_test_keys(
        ooni_exe,
        outfile,
        "-i http://example.org/ web_connectivity",
        "webconnectivity_http_successful_website",
        args,
    )
    assert tk["dns_experiment_failure"] == None
    assert tk["dns_consistency"] == "consistent"
    assert tk["control_failure"] == None
    assert tk["http_experiment_failure"] == None
    assert tk["body_length_match"] == True
    assert tk["body_proportion"] == 1
    assert tk["status_code_match"] == True
    assert tk["headers_match"] == True
    assert tk["title_match"] == True
    assert tk["blocking"] == False
    assert tk["accessible"] == True
    assert_status_flags_are(ooni_exe, tk, 2)


def webconnectivity_https_successful_website(ooni_exe, outfile):
    """ Test case where we succeed with an HTTPS only webpage """
    args = []
    tk = execute_jafar_and_return_validated_test_keys(
        ooni_exe,
        outfile,
        "-i https://example.com/ web_connectivity",
        "webconnectivity_https_successful_website",
        args,
    )
    assert tk["dns_experiment_failure"] == None
    assert tk["dns_consistency"] == "consistent"
    assert tk["control_failure"] == None
    assert tk["http_experiment_failure"] == None
    assert tk["body_length_match"] == True
    assert tk["body_proportion"] == 1
    assert tk["status_code_match"] == True
    assert tk["headers_match"] == True
    assert tk["title_match"] == True
    assert tk["blocking"] == False
    assert tk["accessible"] == True
    assert_status_flags_are(ooni_exe, tk, 1)


def webconnectivity_http_diff_with_inconsistent_dns(ooni_exe, outfile):
    """ Test case where we get an http-diff and the DNS is inconsistent """
    args = [
        "-iptables-hijack-dns-to",
        "127.0.0.1:53",
        "-dns-proxy-hijack",
        "example.org",
        "-http-proxy-block",
        "example.org",
    ]
    tk = execute_jafar_and_return_validated_test_keys(
        ooni_exe,
        outfile,
        "-i http://example.org/ web_connectivity",
        "webconnectivity_http_diff_with_inconsistent_dns",
        args,
    )
    assert tk["dns_experiment_failure"] == None
    assert tk["dns_consistency"] == "inconsistent"
    assert tk["control_failure"] == None
    assert tk["http_experiment_failure"] == None
    assert tk["body_length_match"] == False
    assert tk["body_proportion"] < 1
    assert tk["status_code_match"] == False
    assert tk["headers_match"] == False
    assert tk["title_match"] == False
    assert tk["blocking"] == "dns"
    assert tk["accessible"] == False
    assert_status_flags_are(ooni_exe, tk, 96)


def webconnectivity_http_diff_with_consistent_dns(ooni_exe, outfile):
    """ Test case where we get an http-diff and the DNS is consistent """
    args = [
        "-iptables-hijack-http-to",
        "127.0.0.1:80",
        "-http-proxy-block",
        "example.org",
    ]
    tk = execute_jafar_and_return_validated_test_keys(
        ooni_exe,
        outfile,
        "-i http://example.org/ web_connectivity",
        "webconnectivity_http_diff_with_consistent_dns",
        args,
    )
    assert tk["dns_experiment_failure"] == None
    assert tk["dns_consistency"] == "consistent"
    assert tk["control_failure"] == None
    assert tk["http_experiment_failure"] == None
    assert tk["body_length_match"] == False
    assert tk["body_proportion"] < 1
    assert tk["status_code_match"] == False
    assert tk["headers_match"] == False
    assert tk["title_match"] == False
    assert tk["blocking"] == "http-diff"
    assert tk["accessible"] == False
    assert_status_flags_are(ooni_exe, tk, 64)


def webconnectivity_https_expired_certificate(ooni_exe, outfile):
    """ Test case where the domain's certificate is expired """
    args = []
    tk = execute_jafar_and_return_validated_test_keys(
        ooni_exe,
        outfile,
        "-i https://expired.badssl.com/ web_connectivity",
        "webconnectivity_https_expired_certificate",
        args,
    )
    assert tk["dns_experiment_failure"] == None
    assert tk["dns_consistency"] == "consistent"
    assert tk["control_failure"] == None
    if "miniooni" in ooni_exe:
        assert tk["http_experiment_failure"] == "ssl_invalid_certificate"
    else:
        assert "certificate verify failed" in tk["http_experiment_failure"]
    assert tk["body_length_match"] == None
    assert tk["body_proportion"] == 0
    assert tk["status_code_match"] == None
    assert tk["headers_match"] == None
    assert tk["title_match"] == None
    # The following strikes me as a measurement_kit bug. We are saying
    # that all is good with a domain where actually we don't know why the
    # control is failed and that is clearly not accessible according to
    # our measurement of the domain (certificate expired).
    #
    # See <https://github.com/ooni/probe-engine/issues/858>.
    if "miniooni" in ooni_exe:
        assert tk["blocking"] == None
        assert tk["accessible"] == None
    else:
        assert tk["blocking"] == False
        assert tk["accessible"] == True
    assert_status_flags_are(ooni_exe, tk, 16)


def webconnectivity_https_wrong_host(ooni_exe, outfile):
    """ Test case where the hostname is wrong for the certificate """
    args = []
    tk = execute_jafar_and_return_validated_test_keys(
        ooni_exe,
        outfile,
        "-i https://wrong.host.badssl.com/ web_connectivity",
        "webconnectivity_https_wrong_host",
        args,
    )
    assert tk["dns_experiment_failure"] == None
    assert tk["dns_consistency"] == "consistent"
    assert tk["control_failure"] == None
    if "miniooni" in ooni_exe:
        assert tk["http_experiment_failure"] == "ssl_invalid_hostname"
    else:
        assert "certificate verify failed" in tk["http_experiment_failure"]
    assert tk["body_length_match"] == None
    assert tk["body_proportion"] == 0
    assert tk["status_code_match"] == None
    assert tk["headers_match"] == None
    assert tk["title_match"] == None
    # The following strikes me as a measurement_kit bug. We are saying
    # that all is good with a domain where actually we don't know why the
    # control is failed and that is clearly not accessible according to
    # our measurement of the domain (wrong host for certificate).
    #
    # See <https://github.com/ooni/probe-engine/issues/858>.
    if "miniooni" in ooni_exe:
        assert tk["blocking"] == None
        assert tk["accessible"] == None
    else:
        assert tk["blocking"] == False
        assert tk["accessible"] == True
    assert_status_flags_are(ooni_exe, tk, 16)


def webconnectivity_https_self_signed(ooni_exe, outfile):
    """ Test case where the certificate is self signed """
    args = []
    tk = execute_jafar_and_return_validated_test_keys(
        ooni_exe,
        outfile,
        "-i https://self-signed.badssl.com/ web_connectivity",
        "webconnectivity_https_self_signed",
        args,
    )
    assert tk["dns_experiment_failure"] == None
    assert tk["dns_consistency"] == "consistent"
    assert tk["control_failure"] == None
    if "miniooni" in ooni_exe:
        assert tk["http_experiment_failure"] == "ssl_unknown_authority"
    else:
        assert "certificate verify failed" in tk["http_experiment_failure"]
    assert tk["body_length_match"] == None
    assert tk["body_proportion"] == 0
    assert tk["status_code_match"] == None
    assert tk["headers_match"] == None
    assert tk["title_match"] == None
    # The following strikes me as a measurement_kit bug. We are saying
    # that all is good with a domain where actually we don't know why the
    # control is failed and that is clearly not accessible according to
    # our measurement of the domain (self signed certificate).
    #
    # See <https://github.com/ooni/probe-engine/issues/858>.
    if "miniooni" in ooni_exe:
        assert tk["blocking"] == None
        assert tk["accessible"] == None
    else:
        assert tk["blocking"] == False
        assert tk["accessible"] == True
    assert_status_flags_are(ooni_exe, tk, 16)


def webconnectivity_https_untrusted_root(ooni_exe, outfile):
    """ Test case where the certificate has an untrusted root """
    args = []
    tk = execute_jafar_and_return_validated_test_keys(
        ooni_exe,
        outfile,
        "-i https://untrusted-root.badssl.com/ web_connectivity",
        "webconnectivity_https_untrusted_root",
        args,
    )
    assert tk["dns_experiment_failure"] == None
    assert tk["dns_consistency"] == "consistent"
    assert tk["control_failure"] == None
    if "miniooni" in ooni_exe:
        assert tk["http_experiment_failure"] == "ssl_unknown_authority"
    else:
        assert "certificate verify failed" in tk["http_experiment_failure"]
    assert tk["body_length_match"] == None
    assert tk["body_proportion"] == 0
    assert tk["status_code_match"] == None
    assert tk["headers_match"] == None
    assert tk["title_match"] == None
    # The following strikes me as a measurement_kit bug. We are saying
    # that all is good with a domain where actually we don't know why the
    # control is failed and that is clearly not accessible according to
    # our measurement of the domain (untrusted root certificate).
    #
    # See <https://github.com/ooni/probe-engine/issues/858>.
    if "miniooni" in ooni_exe:
        assert tk["blocking"] == None
        assert tk["accessible"] == None
    else:
        assert tk["blocking"] == False
        assert tk["accessible"] == True
    assert_status_flags_are(ooni_exe, tk, 16)


def webconnectivity_dns_blocking_nxdomain(ooni_exe, outfile):
    """ Test case where there is blocking using NXDOMAIN """
    args = [
        "-iptables-hijack-dns-to",
        "127.0.0.1:53",
        "-dns-proxy-block",
        "example.com",
    ]
    tk = execute_jafar_and_return_validated_test_keys(
        ooni_exe,
        outfile,
        "-i https://example.com/ web_connectivity",
        "webconnectivity_dns_blocking_nxdomain",
        args,
    )
    # The following seems a bug in MK where we don't properly record the
    # actual error that occurred when performing the DNS experiment.
    #
    # See <https://github.com/measurement-kit/measurement-kit/issues/1931>.
    if "miniooni" in ooni_exe:
        assert tk["dns_experiment_failure"] == "dns_nxdomain_error"
    else:
        assert tk["dns_experiment_failure"] == None
    assert tk["dns_consistency"] == "inconsistent"
    assert tk["control_failure"] == None
    if "miniooni" in ooni_exe:
        assert tk["http_experiment_failure"] == None
    else:
        assert tk["http_experiment_failure"] == "dns_lookup_error"
    assert tk["body_length_match"] == None
    assert tk["body_proportion"] == 0
    assert tk["status_code_match"] == None
    assert tk["headers_match"] == None
    assert tk["title_match"] == None
    assert tk["blocking"] == "dns"
    assert tk["accessible"] == False
    assert_status_flags_are(ooni_exe, tk, 2080)


def webconnectivity_https_unknown_authority_with_inconsistent_dns(ooni_exe, outfile):
    """ Test case where the DNS is sending us towards a website where
        we're served an invalid certificate """
    args = [
        "-iptables-hijack-dns-to",
        "127.0.0.1:53",
        "-dns-proxy-hijack",
        "example.org",
        "-bad-proxy-address-tls",
        "127.0.0.1:443",
        "-tls-proxy-address",
        "127.0.0.1:4114",
    ]
    tk = execute_jafar_and_return_validated_test_keys(
        ooni_exe,
        outfile,
        "-i https://example.org/ web_connectivity",
        "webconnectivity_https_unknown_authority_with_inconsistent_dns",
        args,
    )
    assert tk["dns_experiment_failure"] == None
    assert tk["dns_consistency"] == "inconsistent"
    assert tk["control_failure"] == None
    if "miniooni" in ooni_exe:
        assert tk["http_experiment_failure"] == "ssl_unknown_authority"
    else:
        assert "certificate verify failed" in tk["http_experiment_failure"]
    assert tk["body_length_match"] == None
    assert tk["body_proportion"] == 0
    assert tk["status_code_match"] == None
    assert tk["headers_match"] == None
    assert tk["title_match"] == None
    assert tk["blocking"] == "dns"
    assert tk["accessible"] == False
    assert_status_flags_are(ooni_exe, tk, 9248)


def main():
    if len(sys.argv) != 2:
        sys.exit("usage: %s /path/to/ooniprobelegacy-like/binary" % sys.argv[0])
    outfile = "webconnectivity.jsonl"
    ooni_exe = sys.argv[1]
    tests = [
        webconnectivity_https_ok_with_control_failure,
        webconnectivity_http_ok_with_control_failure,
        webconnectivity_transparent_http_proxy,
        webconnectivity_dns_hijacking,
        webconnectivity_control_unreachable_and_using_http,
        webconnectivity_nonexistent_domain,
        webconnectivity_tcpip_blocking_with_consistent_dns,
        webconnectivity_tcpip_blocking_with_inconsistent_dns,
        webconnectivity_http_connection_refused_with_consistent_dns,
        webconnectivity_http_connection_reset_with_consistent_dns,
        webconnectivity_http_nxdomain_with_consistent_dns,
        webconnectivity_http_eof_error_with_consistent_dns,
        webconnectivity_http_generic_timeout_error_with_consistent_dns,
        webconnectivity_http_connection_reset_with_inconsistent_dns,
        webconnectivity_http_successful_website,
        webconnectivity_https_successful_website,
        webconnectivity_http_diff_with_inconsistent_dns,
        webconnectivity_http_diff_with_consistent_dns,
        webconnectivity_https_expired_certificate,
        webconnectivity_https_wrong_host,
        webconnectivity_https_self_signed,
        webconnectivity_https_untrusted_root,
        webconnectivity_dns_blocking_nxdomain,
        webconnectivity_https_unknown_authority_with_inconsistent_dns,
    ]
    for test in tests:
        test(ooni_exe, outfile)
        time.sleep(7)


if __name__ == "__main__":
    main()
