// This script performs QA checks.

import { runTestCase } from "./lib/runner.mjs"

// hijackPopularDNSServers returns an object containing the rules
// for hijacking popular DNS servers with `miniooni --censor`.
//
// This function is an helper function for populating test cases.
function hijackPopularDNSServers() {
    return {
        // cloudflare
        "1.1.1.1:53/udp": "hijack-dns",
        "1.0.0.1:53/udp": "hijack-dns",
        // google
        "8.8.8.8:53/udp": "hijack-dns",
        "8.8.4.4:53/udp": "hijack-dns",
        // quad9
        "9.9.9.9:53/udp": "hijack-dns",
        "9.9.9.10:53/udp": "hijack-dns",
    }
}

// checkExperimentName ensures that the experimentName is correctly
// set in the output report file.
//
// This function helps to implement per-experiment checkers.
function checkExperimentName(name, report) {
    const result = report.test_name === name
    console.log(`checking whether the experiment name is correct... ${result}`)
    return result
}

// checkTopLevelKeys is a function that invokes a standard set of checks
// to validate the top-level test keys of a result.
//
// This function helps to implement per-experiment checkers.
function checkTopLevelKeys(name, report) {
    let result = true
    result = result && checkExperimentName(name, report)
    return result
}

// testCases lists all the test cases.
//
// A test case is an object with the following fields:
//
// - name (string): the name of the test case;
//
// - description (string): a description of the test case;
//
// - input (string): the input to pass to the experiment;
//
// - blocking (object): a blocking specification (i.e., the
// serialization of a filtering.TProxyConfig struct);
//
// - experiments (object): names of the experiments to run
// mapping to the function to use to verify the results.
const testCases = [{
    name: "web_dns_nxdomain",
    description: "nxdomain for the domain inside the URL",
    input: "https://nexa.polito.it/",
    blocking: {
        Domains: {
            "nexa.polito.it": "nxdomain",
        },
        Endpoints: hijackPopularDNSServers(),
    },
    experiments: {
        websteps: (name, report) => {
            return checkTopLevelKeys(name, report)
        },
        web_connectivity: (name, report) => {
            return checkTopLevelKeys(name, report)
        },
    },
},

{
    name: "web_dns_refused",
    description: "refused for the domain inside the URL",
    input: "https://nexa.polito.it/",
    blocking: {
        Domains: {
            "nexa.polito.it": "refused",
        },
        Endpoints: hijackPopularDNSServers(),
    },
    experiments: {
        websteps: (name, report) => {
            return checkTopLevelKeys(name, report)
        },
        web_connectivity: (name, report) => {
            return checkTopLevelKeys(name, report)
        },
    },
},

{
    name: "web_dns_localhost",
    description: "localhost reply for the domain inside the URL",
    input: "https://nexa.polito.it/",
    blocking: {
        Domains: {
            "nexa.polito.it": "localhost",
        },
        Endpoints: hijackPopularDNSServers(),
    },
    experiments: {
        websteps: (name, report) => {
            return checkTopLevelKeys(name, report)
        },
        web_connectivity: (name, report) => {
            return checkTopLevelKeys(name, report)
        },
    },
},

{
    name: "web_dns_no_answer",
    description: "no answer for the domain inside the URL",
    input: "https://nexa.polito.it/",
    blocking: {
        Domains: {
            "nexa.polito.it": "no-answer",
        },
        Endpoints: hijackPopularDNSServers(),
    },
    experiments: {
        websteps: (name, report) => {
            return checkTopLevelKeys(name, report)
        },
        web_connectivity: (name, report) => {
            return checkTopLevelKeys(name, report)
        },
    },
},

{
    name: "web_dns_timeout",
    description: "timeout when resolving the domain inside the URL",
    input: "https://nexa.polito.it/",
    blocking: {
        Domains: {
            "nexa.polito.it": "timeout",
        },
        Endpoints: hijackPopularDNSServers(),
    },
    experiments: {
        websteps: (name, report) => {
            return checkTopLevelKeys(name, report)
        },
        web_connectivity: (name, report) => {
            return checkTopLevelKeys(name, report)
        },
    },

},

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
        websteps: (name, report) => {
            return checkTopLevelKeys(name, report)
        },
        web_connectivity: (name, report) => {
            return checkTopLevelKeys(name, report)
        },
    },
},

{
    name: "web_tcp_connection_refused",
    description: "connection refused when connecting to the IP address",
    input: "https://nexa.polito.it/",
    blocking: {
        Endpoints: {
            "130.192.16.171:443/tcp": "tcp-reject-syn",
        },
    },
    experiments: {
        websteps: (name, report) => {
            return checkTopLevelKeys(name, report)
        },
        web_connectivity: (name, report) => {
            return checkTopLevelKeys(name, report)
        },
    },
},

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
        websteps: (name, report) => {
            return checkTopLevelKeys(name, report)
        },
        web_connectivity: (name, report) => {
            return checkTopLevelKeys(name, report)
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
        websteps: (name, report) => {
            return checkTopLevelKeys(name, report)
        },
        web_connectivity: (name, report) => {
            return checkTopLevelKeys(name, report)
        },
    },
},

{
    name: "web_tls_handshake_validation",
    description: "validation issues when performing the TLS handshake",
    input: "https://nexa.polito.it/",
    blocking: {
        Endpoints: {
            "130.192.16.171:443/tcp": "divert",
        },
        Divert: {
            "130.192.16.171:443/tcp": "8.8.4.4:443/tcp",
        }
    },
    experiments: {
        websteps: (name, report) => {
            return checkTopLevelKeys(name, report)
        },
        web_connectivity: (name, report) => {
            return checkTopLevelKeys(name, report)
        },
    },

},

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
        websteps: (name, report) => {
            return checkTopLevelKeys(name, report)
        },
    },
},

{
    name: "web_quic_handshake_validation",
    description: "validation error when performing the QUIC handshake",
    input: "https://dns.google/",
    blocking: {
        Endpoints: {
            "8.8.8.8:443/udp": "divert",
            "8.8.4.4:443/udp": "divert",
            "[2001:4860:4860::8888]:443/udp": "divert",
            "[2001:4860:4860::8844]:443/udp": "divert",
        },
        Divert: {
            "8.8.8.8:443/udp": "1.1.1.1:443/udp",
            "8.8.4.4:443/udp": "1.1.1.1:443/udp",
            "[2001:4860:4860::8888]:443/udp": "1.1.1.1:443/udp",
            "[2001:4860:4860::8844]:443/udp": "1.1.1.1:443/udp",
        }
    },
    experiments: {
        websteps: (name, report) => {
            return checkTopLevelKeys(name, report)
        },
    },
},

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
        websteps: (name, report) => {
            return checkTopLevelKeys(name, report)
        },
        web_connectivity: (name, report) => {
            return checkTopLevelKeys(name, report)
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
        websteps: (name, report) => {
            return checkTopLevelKeys(name, report)
        },
        web_connectivity: (name, report) => {
            return checkTopLevelKeys(name, report)
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
        websteps: (name, report) => {
            return checkTopLevelKeys(name, report)
        },
        web_connectivity: (name, report) => {
            return checkTopLevelKeys(name, report)
        },
    },
},

// TODO: we need support for this functionality
/*
{
    name: "0015_web_http_403",
    description: "403 when performing the HTTPS round trip",
},
*/

]

// checkForDuplicateTestCases ensures there are no duplicate names.
function checkForDuplicateTestCases() {
    let dups = {}
    for (let i = 0; i < testCases.length; i++) {
        const testCase = testCases[i]
        if (dups[testCase.name] !== undefined) {
            console.log(`fatal: duplicate test case name: ${testCase.name}`)
            process.exit(1)
        }
        dups[testCase.name] = true
    }
}

// runAllTestCases runs all the available test cases.
function runAllTestCases() {
    let result = true
    for (let i = 0; i < testCases.length; i++) {
        result = result && runTestCase(testCases[i])
    }
    return result
}

// makeTestCasesMap creates a map from the test case name
// to the test case definition.
function makeTestCasesMap() {
    var map = {}
    for (let i = 0; i < testCases.length; i++) {
        const testCase = testCases[i]
        map[testCase.name] = testCase
    }
    return map
}

// commandRun implements the run command.
function commandRun(args) {
    const bailOnFailure = (result) => {
        if (!result) {
            console.log("some checks failed (see above logs)")
            process.exit(1)
        }
    }
    if (args.length < 1) {
        bailOnFailure(runAllTestCases())
        return
    }
    let result = true
    const map = makeTestCasesMap()
    for (let i = 0; i < args.length; i++) {
        const arg = args[i]
        const testCase = map[arg]
        if (testCase === undefined) {
            console.log(`unknown test case: ${arg}`)
            process.exit(1)
        }
        result = result && runTestCase(testCase)
    }
    bailOnFailure(result)
}

// commandList implements the list command.
function commandList() {
    for (let i = 0; i < testCases.length; i++) {
        const testCase = testCases[i]
        console.log(`${testCase.name}:`)
        console.log(`\t${testCase.description}`)
    }
}

// main is the main function.
function main() {
    const usageAndExit = (exitcode) => {
        console.log("usage: node ./QA/web.mjs list")
        console.log("usage: node ./QA/web.mjs run [test_case_name...]")
        process.exit(exitcode)
    }
    if (process.argv.length < 3) {
        usageAndExit(0)
    }
    checkForDuplicateTestCases()
    const command = process.argv[2]
    switch (command) {
        case "list":
            commandList()
            break
        case "run":
            commandRun(process.argv.slice(3))
            break
        default:
            console.log(`unknown command: ${command}`)
            usageAndExit(1)
    }
}

main()
