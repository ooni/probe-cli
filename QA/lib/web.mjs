// This file contains test cases for web nettests.

import { checkMeasurement } from "./analysis.mjs"

// webConnectivityCheckTopLevel checks the top-level keys
// of the web connectivity experiment agains a template
// object. Returns true if they match, false on mismatch.
function webConnectivityCheckTopLevel(tk, template) {
    let result = true
    for (const [key, value] of Object.entries(template)) {
        const check = tk[key] === value
        console.log(`checking whether ${key}'s value is ${value}... ${check}`)
        result = result && check
    }
    return result
}

// Here we export all the test cases. Please, see the documentation
// of runner.runTestCase for a description of what is a test case.
export const testCases = [

    //
    // DNS checks
    //
    // We start with checks where _only_ the system resolver fails.
    //

    {
        name: "web_dns_system_nxdomain",
        description: "the system resolver returns NXDOMAIN",
        input: "https://nexa.polito.it/",
        blocking: {
            Domains: {
                "nexa.polito.it": "nxdomain"
            },
        },
        experiments: {
            websteps: (testCase, name, report) => {
                return checkMeasurement(testCase, name, report)
            },
            web_connectivity: (testCase, name, report) => {
                return checkMeasurement(testCase, name, report, (tk) => {
                    let result = true
                    result = result && webConnectivityCheckTopLevel(tk, {
                        "dns_experiment_failure": "dns_nxdomain_error",
                        "dns_consistency": "inconsistent",
                        "control_failure": null,
                        "body_length_match": null,
                        "body_proportion": 0,
                        "status_code_match": null,
                        "headers_match": null,
                        "title_match": null,
                        "accessible": false,
                        "blocking": "dns",
                    })
                    return result
                })
            },
        },
    },

    {
        name: "web_dns_system_refused",
        description: "the system resolver returns REFUSED",
        input: "https://nexa.polito.it/",
        blocking: {
            Domains: {
                "nexa.polito.it": "refused"
            },
        },
        experiments: {
            websteps: (testCase, name, report) => {
                return checkMeasurement(testCase, name, report)
            },
            web_connectivity: (testCase, name, report) => {
                return checkMeasurement(testCase, name, report, (tk) => {
                    let result = true
                    result = result && webConnectivityCheckTopLevel(tk, {
                        "dns_experiment_failure": "dns_refused_error",
                        "dns_consistency": "inconsistent",
                        "control_failure": null,
                        "body_length_match": null,
                        "body_proportion": 0,
                        "status_code_match": null,
                        "headers_match": null,
                        "title_match": null,
                        "accessible": null,
                        "blocking": null, // TODO(bassosimone): this is clearly a bug
                    })
                    return result
                })
            },
        },
    },

    {
        name: "web_dns_system_localhost",
        description: "the system resolver returns localhost",
        input: "https://nexa.polito.it/",
        blocking: {
            Domains: {
                "nexa.polito.it": "localhost",
            },
        },
        experiments: {
            websteps: (testCase, name, report) => {
                return checkMeasurement(testCase, name, report)
            },
            web_connectivity: (testCase, name, report) => {
                return checkMeasurement(testCase, name, report, (tk) => {
                    let result = true
                    // TODO(bassosimone): Web Connectivity does not correctly handle this case
                    // but, still, correctly sets blocking as "dns"
                    result = result && webConnectivityCheckTopLevel(tk, {
                        "dns_experiment_failure": null,
                        "dns_consistency": "inconsistent",
                        "control_failure": null,
                        "http_experiment_failure": "connection_refused",
                        "body_length_match": null,
                        "body_proportion": 0,
                        "status_code_match": null,
                        "headers_match": null,
                        "title_match": null,
                        "accessible": false,
                        "blocking": "dns",
                    })
                    return result
                })
            },
        },
    },

    {
        name: "web_dns_system_bogon_not_localhost",
        description: "the system resolver returns a bogon that is not localhost",
        input: "https://nexa.polito.it/",
        blocking: {
            DNSCache: {
                "nexa.polito.it": ["10.0.0.1"],
            },
            Domains: {
                "nexa.polito.it": "cache",
            },
        },
        experiments: {
            websteps: (testCase, name, report) => {
                return checkMeasurement(testCase, name, report)
            },
            web_connectivity: (testCase, name, report) => {
                return checkMeasurement(testCase, name, report, (tk) => {
                    let result = true
                    // TODO(bassosimone): Web Connectivity does not correctly handle this case
                    // but, still, correctly sets blocking as "dns"
                    result = result && webConnectivityCheckTopLevel(tk, {
                        "dns_experiment_failure": null,
                        "dns_consistency": "inconsistent",
                        "control_failure": null,
                        "http_experiment_failure": "generic_timeout_error",
                        "body_length_match": null,
                        "body_proportion": 0,
                        "status_code_match": null,
                        "headers_match": null,
                        "title_match": null,
                        "accessible": false,
                        "blocking": "dns",
                    })
                    return result
                })
            },
        },
    },

    {
        name: "web_dns_system_no_answer",
        description: "the system resolver returns an empty answer",
        input: "https://nexa.polito.it/",
        blocking: {
            Domains: {
                "nexa.polito.it": "no-answer",
            },
        },
        experiments: {
            websteps: (testCase, name, report) => {
                return checkMeasurement(testCase, name, report)
            },
            web_connectivity: (testCase, name, report) => {
                return checkMeasurement(testCase, name, report, (tk) => {
                    let result = true
                    result = result && webConnectivityCheckTopLevel(tk, {
                        "dns_experiment_failure": "dns_no_answer",
                        "dns_consistency": "inconsistent",
                        "control_failure": null,
                        "http_experiment_failure": null,
                        "body_length_match": null,
                        "body_proportion": 0,
                        "status_code_match": null,
                        "headers_match": null,
                        "title_match": null,
                        "accessible": null,
                        "blocking": null, // TODO(bassosimone): this is clearly a bug
                    })
                    return result
                })
            },
        },
    },

    {
        name: "web_dns_system_timeout",
        description: "the system resolver times out",
        input: "https://nexa.polito.it/",
        blocking: {
            Domains: {
                "nexa.polito.it": "timeout",
            },
        },
        experiments: {
            websteps: (testCase, name, report) => {
                return checkMeasurement(testCase, name, report)
            },
            web_connectivity: (testCase, name, report) => {
                return checkMeasurement(testCase, name, report, (tk) => {
                    let result = true
                    result = result && webConnectivityCheckTopLevel(tk, {
                        "dns_experiment_failure": "generic_timeout_error",
                        "dns_consistency": "inconsistent",
                        "control_failure": null,
                        "http_experiment_failure": null,
                        "body_length_match": null,
                        "body_proportion": 0,
                        "status_code_match": null,
                        "headers_match": null,
                        "title_match": null,
                        "accessible": null,
                        "blocking": null, // TODO(bassosimone): this is clearly a bug
                    })
                    return result
                })
            },
        },

    },

    // TODO(bassosimone): here we should insert more checks where not only the system
    // resolver is blocked but also other resolvers are.

    //
    // TCP connect
    //
    // This section contains TCP connect failures.
    //

    {
        name: "web_tcp_connect_timeout",
        description: "timeout when connecting to the IP address",
        input: "https://nexa.polito.it/",
        blocking: {
            Endpoints: {
                "130.192.16.171:443/tcp": "tcp-drop-syn",
            },
        },
        experiments: {
            websteps: (testCase, name, report) => {
                return checkMeasurement(testCase, name, report)
            },
            web_connectivity: (testCase, name, report) => {
                return checkMeasurement(testCase, name, report, (tk) => {
                    let result = true
                    result = result && webConnectivityCheckTopLevel(tk, {
                        "dns_experiment_failure": null,
                        "dns_consistency": "consistent",
                        "control_failure": null,
                        "http_experiment_failure": "generic_timeout_error",
                        "body_length_match": null,
                        "body_proportion": 0,
                        "status_code_match": null,
                        "headers_match": null,
                        "title_match": null,
                        "accessible": false,
                        "blocking": "tcp_ip",
                    })
                    return result
                })
            },
        },
    },

    {
        name: "web_tcp_connect_refused",
        description: "connection refused when connecting to the IP address",
        input: "https://nexa.polito.it/",
        blocking: {
            Endpoints: {
                "130.192.16.171:443/tcp": "tcp-reject-syn",
            },
        },
        experiments: {
            websteps: (testCase, name, report) => {
                return checkMeasurement(testCase, name, report)
            },
            web_connectivity: (testCase, name, report) => {
                return checkMeasurement(testCase, name, report, (tk) => {
                    let result = true
                    result = result && webConnectivityCheckTopLevel(tk, {
                        "dns_experiment_failure": null,
                        "dns_consistency": "consistent",
                        "control_failure": null,
                        "http_experiment_failure": "connection_refused",
                        "body_length_match": null,
                        "body_proportion": 0,
                        "status_code_match": null,
                        "headers_match": null,
                        "title_match": null,
                        "accessible": false,
                        "blocking": "tcp_ip",
                    })
                    return result
                })
            },
        },
    },

    //
    // TLS handshake
    //
    // This section contains TLS handshake failures.
    //

    {
        name: "web_tls_handshake_timeout",
        description: "timeout when performing the TLS handshake",
        input: "https://nexa.polito.it/",
        blocking: {
            Endpoints: {
                "130.192.16.171:443/tcp": "drop-data",
            },
        },
        experiments: {
            websteps: (testCase, name, report) => {
                return checkMeasurement(testCase, name, report)
            },
            web_connectivity: (testCase, name, report) => {
                return checkMeasurement(testCase, name, report, (tk) => {
                    let result = true
                    result = result && webConnectivityCheckTopLevel(tk, {
                        "dns_experiment_failure": null,
                        "dns_consistency": "consistent",
                        "control_failure": null,
                        "http_experiment_failure": "generic_timeout_error",
                        "body_length_match": null,
                        "body_proportion": 0,
                        "status_code_match": null,
                        "headers_match": null,
                        "title_match": null,
                        "accessible": false,
                        "blocking": "http-failure",
                    })
                    return result
                })
            },
        },
    },

    {
        name: "web_tls_handshake_reset",
        description: "reset when performing the TLS handshake",
        input: "https://nexa.polito.it/",
        blocking: {
            Endpoints: {
                "130.192.16.171:443/tcp": "hijack-tls",
            },
            SNIs: {
                "nexa.polito.it": "reset",
            },
        },
        experiments: {
            websteps: (testCase, name, report) => {
                return checkMeasurement(testCase, name, report)
            },
            web_connectivity: (testCase, name, report) => {
                return checkMeasurement(testCase, name, report, (tk) => {
                    let result = true
                    result = result && webConnectivityCheckTopLevel(tk, {
                        "dns_experiment_failure": null,
                        "dns_consistency": "consistent",
                        "control_failure": null,
                        "http_experiment_failure": "connection_reset",
                        "body_length_match": null,
                        "body_proportion": 0,
                        "status_code_match": null,
                        "headers_match": null,
                        "title_match": null,
                        "accessible": false,
                        "blocking": "http-failure",
                    })
                    return result
                })
            },
        },
    },

    //
    // QUIC
    //

    {
        name: "web_quic_handshake_timeout",
        description: "timeout when performing the QUIC handshake",
        input: "https://dns.google/",
        blocking: {
            Endpoints: {
                "8.8.8.8:443/udp": "drop-data",
                "8.8.4.4:443/udp": "drop-data",
                "[2001:4860:4860::8888]:443/udp": "drop-data",
                "[2001:4860:4860::8844]:443/udp": "drop-data",
            },
        },
        experiments: {
            websteps: (testCase, name, report) => {
                return checkMeasurement(testCase, name, report)
            },
        },
    },

    //
    // Cleartext HTTP
    //

    {
        name: "web_http_reset",
        description: "reset when performing the HTTP round trip",
        input: "http://nexa.polito.it/",
        blocking: {
            Endpoints: {
                "130.192.16.171:80/tcp": "hijack-http",
            },
            Hosts: {
                "nexa.polito.it": "reset",
            },
        },
        experiments: {
            websteps: (testCase, name, report) => {
                return checkMeasurement(testCase, name, report)
            },
            web_connectivity: (testCase, name, report) => {
                return checkMeasurement(testCase, name, report, (tk) => {
                    let result = true
                    result = result && webConnectivityCheckTopLevel(tk, {
                        "dns_experiment_failure": null,
                        "dns_consistency": "consistent",
                        "control_failure": null,
                        "http_experiment_failure": "connection_reset",
                        "body_length_match": null,
                        "body_proportion": 0,
                        "status_code_match": null,
                        "headers_match": null,
                        "title_match": null,
                        "accessible": false,
                        "blocking": "http-failure",
                    })
                    return result
                })
            },
        },
    },

    {
        name: "web_http_timeout",
        description: "timeout when performing the HTTP round trip",
        input: "http://nexa.polito.it/",
        blocking: {
            Endpoints: {
                "130.192.16.171:80/tcp": "hijack-http",
            },
            Hosts: {
                "nexa.polito.it": "timeout",
            },
        },
        experiments: {
            websteps: (testCase, name, report) => {
                return checkMeasurement(testCase, name, report)
            },
            web_connectivity: (testCase, name, report) => {
                return checkMeasurement(testCase, name, report, (tk) => {
                    let result = true
                    result = result && webConnectivityCheckTopLevel(tk, {
                        "dns_experiment_failure": null,
                        "dns_consistency": "consistent",
                        "control_failure": null,
                        "http_experiment_failure": "generic_timeout_error",
                        "body_length_match": null,
                        "body_proportion": 0,
                        "status_code_match": null,
                        "headers_match": null,
                        "title_match": null,
                        "accessible": false,
                        "blocking": "http-failure",
                    })
                    return result
                })
            },
        },
    },

    {
        name: "web_http_451",
        description: "451 when performing the HTTP round trip",
        input: "http://nexa.polito.it/",
        blocking: {
            Endpoints: {
                "130.192.16.171:80/tcp": "hijack-http",
            },
            Hosts: {
                "nexa.polito.it": "451",
            },
        },
        experiments: {
            websteps: (testCase, name, report) => {
                return checkMeasurement(testCase, name, report)
            },
            web_connectivity: (testCase, name, report) => {
                return checkMeasurement(testCase, name, report, (tk) => {
                    let result = true
                    // TODO(bassosimone): there is no easy way to check for the body
                    // proportion robustly because it's a float.
                    result = result && webConnectivityCheckTopLevel(tk, {
                        "dns_experiment_failure": null,
                        "dns_consistency": "consistent",
                        "control_failure": null,
                        "http_experiment_failure": null,
                        "body_length_match": false,
                        "status_code_match": false,
                        "headers_match": false,
                        "title_match": false,
                        "accessible": false,
                        "blocking": "http-diff",
                    })
                    return result
                })
            },
        },
    },

    //
    // More complex scenarios
    //

    // In this scenario the second IP address for the domain fails
    // with reset. Web Connectivity sees that but overall says it's
    // all good because the good IP happens to be the first. We'll
    // see what changes if we swap the IPs in the next scenario.
    {
        name: "web_tcp_second_ip_connection_reset",
        description: "the second IP returned by DNS fails with connection reset",
        input: "https://dns.google/",
        blocking: {
            DNSCache: {
                "dns.google": ["8.8.4.4", "8.8.8.8"],
            },
            Domains: {
                "dns.google": "cache",
            },
            Endpoints: {
                "8.8.8.8:443/tcp": "hijack-tls",
            },
            SNIs: {
                "dns.google": "reset",
            },
        },
        experiments: {
            websteps: (testCase, name, report) => {
                return checkMeasurement(testCase, name, report)
            },
            web_connectivity: (testCase, name, report) => {
                return checkMeasurement(testCase, name, report, (tk) => {
                    let result = true
                    result = result && webConnectivityCheckTopLevel(tk, {
                        "dns_experiment_failure": null,
                        "dns_consistency": "consistent",
                        "control_failure": null,
                        "http_experiment_failure": null,
                        "body_length_match": true,
                        "body_proportion": 1,
                        "status_code_match": true,
                        "headers_match": true,
                        "title_match": true,
                        "accessible": true,
                        "blocking": false,
                    })
                    return result
                })
            },
        },
    },

    // This scenario is like the previous one except that we swap
    // the IP addresses and now Web Connectivity says failure.
    {
        name: "web_tcp_first_ip_connection_reset",
        description: "the first IP returned by DNS fails with connection reset",
        input: "https://dns.google/",
        blocking: {
            DNSCache: {
                "dns.google": ["8.8.4.4", "8.8.8.8"],
            },
            Domains: {
                "dns.google": "cache",
            },
            Endpoints: {
                "8.8.4.4:443/tcp": "hijack-tls",
            },
            SNIs: {
                "dns.google": "reset",
            },
        },
        experiments: {
            websteps: (testCase, name, report) => {
                return checkMeasurement(testCase, name, report)
            },
            web_connectivity: (testCase, name, report) => {
                return checkMeasurement(testCase, name, report, (tk) => {
                    let result = true
                    result = result && webConnectivityCheckTopLevel(tk, {
                        "dns_experiment_failure": null,
                        "dns_consistency": "consistent",
                        "control_failure": null,
                        "http_experiment_failure": "connection_reset",
                        "body_length_match": null,
                        "body_proportion": 0,
                        "status_code_match": null,
                        "headers_match": null,
                        "title_match": null,
                        "accessible": false,
                        "blocking": "http-failure",
                    })
                    return result
                })
            },
        },
    },
]
