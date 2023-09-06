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
