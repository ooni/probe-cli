#!/usr/bin/env python3


""" ./QA/webconnectivity.py - main QA script for webconnectivity

    This script performs a bunch of webconnectivity tests under censored
    network conditions and verifies that the measurement is consistent
    with the expectations, by parsing the resulting JSONL. """

import socket
import sys
import time

sys.path.insert(0, ".")
import common


def execute_jafar_and_return_validated_test_keys(
    ooni_exe, outfile, experiment_args, tag, args
):
    """Executes jafar and returns the validated parsed test keys, or throws
    an AssertionError if the result is not valid."""
    tk = common.execute_jafar_and_miniooni(
        ooni_exe, outfile, experiment_args, tag, args
    )
    print("dns_experiment_failure", tk["dns_experiment_failure"], file=sys.stderr)
    print("dns_consistency", tk["dns_consistency"], file=sys.stderr)
    print("control_failure", tk["control_failure"], file=sys.stderr)
    print("http_experiment_failure", tk["http_experiment_failure"], file=sys.stderr)
    print("tk_body_length_match", tk["body_length_match"], file=sys.stderr)
    print("body_proportion", tk["body_proportion"], file=sys.stderr)
    print("status_code_match", tk["status_code_match"], file=sys.stderr)
    print("headers_match", tk["headers_match"], file=sys.stderr)
    print("title_match", tk["title_match"], file=sys.stderr)
    print("blocking", tk["blocking"], file=sys.stderr)
    print("accessible", tk["accessible"], file=sys.stderr)
    print("x_status", tk["x_status"], file=sys.stderr)
    return tk


def assert_status_flags_are(ooni_exe, tk, desired):
    """Checks whether the status flags are what we expect them to
    be when we're running miniooni. This check only makes sense
    with miniooni b/c status flags are a miniooni extension."""
    if "miniooni" not in ooni_exe:
        return
    assert tk["x_status"] == desired


def webconnectivity_https_ok_with_control_failure(ooni_exe, outfile):
    """Successful HTTPS measurement but control failure."""
    # Note: this QA check will increasingly become more difficult to implement
    # as we continue to improve our fallback TH strategies
    args = [
        "-iptables-reset-keyword",
        "th.ooni.org",
        "-iptables-reset-keyword",
        "d33d1gs9kpq1c5.cloudfront.net",
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
    assert "connection_reset" in tk["control_failure"]
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
    """Successful HTTP measurement but control failure."""
    # Note: this QA check will increasingly become more difficult to implement
    # as we continue to improve our fallback TH strategies
    args = [
        "-iptables-reset-keyword",
        "th.ooni.org",
        "-iptables-reset-keyword",
        "d33d1gs9kpq1c5.cloudfront.net",
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
    assert "connection_reset" in tk["control_failure"]
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
    """Test case where we pass through a transparent HTTP proxy"""
    args = []
    args.append("-iptables-hijack-http-to")
    args.append("127.0.0.1:80")
    tk = execute_jafar_and_return_validated_test_keys(
        ooni_exe,
        outfile,
        "-i http://example.org web_connectivity",
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
    assert_status_flags_are(ooni_exe, tk, 2)


def webconnectivity_transparent_https_proxy(ooni_exe, outfile):
    """Test case where we pass through a transparent HTTPS proxy"""
    args = []
    args.append("-iptables-hijack-https-to")
    args.append("127.0.0.1:443")
    tk = execute_jafar_and_return_validated_test_keys(
        ooni_exe,
        outfile,
        "-i https://example.org web_connectivity",
        "webconnectivity_transparent_https_proxy",
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
    """Test case where there is DNS hijacking towards a transparent proxy."""
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


def webconnectivity_http_connection_refused_with_consistent_dns(ooni_exe, outfile):
    """Test case where there's TCP/IP blocking w/ consistent DNS that occurs
    while we're following the chain of redirects."""
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
    """Test case where there's RST-based blocking blocking w/ consistent DNS that
    occurs while we're following the chain of redirects."""
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
    """Test case where there's a redirection and the redirected request cannot
    continue because a NXDOMAIN error occurs."""
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
    """Test case where there's a redirection and the redirected request cannot
    continue because an eof_error error occurs."""
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
    """Test case where there's a redirection and the redirected request cannot
    continue because a generic_timeout_error error occurs."""
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
    """Test case where there's inconsistent DNS and the connection is RST when
    we're executing HTTP code."""
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


def webconnectivity_http_diff_with_inconsistent_dns(ooni_exe, outfile):
    """Test case where we get an http-diff and the DNS is inconsistent"""
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
    assert tk["headers_match"] == True
    assert tk["title_match"] == False
    assert tk["blocking"] == "dns"
    assert tk["accessible"] == False
    assert_status_flags_are(ooni_exe, tk, 96)


def webconnectivity_http_diff_with_consistent_dns(ooni_exe, outfile):
    """Test case where we get an http-diff and the DNS is consistent"""
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
    assert tk["headers_match"] == True
    assert tk["title_match"] == False
    assert tk["blocking"] == "http-diff"
    assert tk["accessible"] == False
    assert_status_flags_are(ooni_exe, tk, 64)


def webconnectivity_https_expired_certificate(ooni_exe, outfile):
    """Test case where the domain's certificate is expired"""
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
    """Test case where the hostname is wrong for the certificate"""
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
    """Test case where the certificate is self signed"""
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
    """Test case where the certificate has an untrusted root"""
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


def webconnectivity_https_unknown_authority_with_inconsistent_dns(ooni_exe, outfile):
    """Test case where the DNS is sending us towards a website where
    we're served an invalid certificate"""
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
        webconnectivity_transparent_https_proxy,
        webconnectivity_dns_hijacking,
        webconnectivity_http_connection_refused_with_consistent_dns,
        webconnectivity_http_connection_reset_with_consistent_dns,
        webconnectivity_http_nxdomain_with_consistent_dns,
        webconnectivity_http_eof_error_with_consistent_dns,
        webconnectivity_http_generic_timeout_error_with_consistent_dns,
        webconnectivity_http_connection_reset_with_inconsistent_dns,
        webconnectivity_http_diff_with_inconsistent_dns,
        webconnectivity_http_diff_with_consistent_dns,
        webconnectivity_https_expired_certificate,
        webconnectivity_https_wrong_host,
        webconnectivity_https_self_signed,
        webconnectivity_https_untrusted_root,
        webconnectivity_https_unknown_authority_with_inconsistent_dns,
    ]
    for test in tests:
        test(ooni_exe, outfile)
        time.sleep(7)


if __name__ == "__main__":
    main()
