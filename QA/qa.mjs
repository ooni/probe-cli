// This script performs QA checks.

import child_process from "child_process"
import crypto from "crypto"
import fs from "fs"
import path from "path"

// hijackPopularDNSServers returns an object containing the rules
// for hijacking popular DNS servers with `miniooni --censor`.
//
// This function is an helper function for populating test cases.
const hijackPopularDNSServers = () => {
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
const checkExperimentName = (name, report) => {
    const result = report.test_name === name
    console.log(`checking whether the experiment name is correct... ${result}`)
    return result
}

// checkTopLevelKeys is a function that invokes a standard set of checks
// to validate the top-level test keys of a result.
//
// This function helps to implement per-experiment checkers.
const checkTopLevelKeys = (name, report) => {
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
    name: "0000_web_dns_nxdomain",
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
    name: "0001_web_dns_refused",
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
    name: "0002_web_dns_bogon",
    description: "bogon reply for the domain inside the URL",
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
    name: "0003_web_dns_no_answer",
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
    name: "0004_web_dns_timeout",
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
    name: "0005_web_tcp_connect_timeout",
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
    name: "0006_web_tcp_connection_refused",
    description: "connection refused when connecting to the IP address",
    input: "https://nexa.polito.it/",
    blocking: {
        Endpoints: {
            "130.192.16.171:443/tcp": "tcp-reject",
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
    name: "0007_web_tls_handshake_timeout",
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
    name: "0008_web_tls_handshake_reset",
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
    name: "0009_web_tls_handshake_validation",
    description: "validation issues when performing the TLS handshake",
    input: "https://nexa.polito.it/",
    blocking: {
        Endpoints: {
            "130.192.16.171:443/tcp": "hijack-tls-mitm",
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
    name: "0010_web_quic_handshake_timeout",
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
    name: "0011_web_quic_handshake_validation",
    description: "validation error when performing the QUIC handshake",
    input: "https://dns.google/",
    blocking: {
        Endpoints: {
            "8.8.8.8:443/udp": "hijack-quic-mitm",
            "8.8.4.4:443/udp": "hijack-quic-mitm",
            "[2001:4860:4860::8888]:443/udp": "hijack-quic-mitm",
            "[2001:4860:4860::8844]:443/udp": "hijack-quic-mitm",
        },
    },
    experiments: {
        websteps: (name, report) => {
            return checkTopLevelKeys(name, report)
        },
    },
},

{
    name: "0012_web_http_reset",
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
    name: "0013_web_http_timeout",
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
    name: "0014_web_http_451",
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

// tempFile returns the name of a temporary file. This function is
// not as secure as using mktemp but it does not matter in this
// context. We just need to create file names in the local directory
// with enough entropy that every run has a different name.
//
// See https://stackoverflow.com/questions/7055061 for an insightful
// discussion on how one should or should not create temp files.
const tempFile = (suffix) => {
    return path.join(`tmp-${crypto.randomBytes(16).toString('hex')}.${suffix}`);
}

// exec executes a command. This function throws on failure.
const exec = (command) => {
    console.log(`+ ${command}`)
    child_process.execSync(command)
}

// writeCensorJsonFile writes a censor.json file using a file name
// containing random characters and returns the file name.
const writeCensorJsonFile = (testCase) => {
    const fileName = tempFile("json")
    fs.writeFileSync(fileName, JSON.stringify(testCase.blocking))
    return fileName
}

// readReportFile reads and parses the report file, thus returning
// the JSON object contained inside the report file.
const readReportFile = (reportFile) => {
    const data = fs.readFileSync(reportFile, { "encoding": "utf-8" })
    return JSON.parse(data)
}

// runExperiment runs the given test case with the given experiment.
const runExperiment = (testCase, experiment, checker) => {
    console.log(`## running: ${testCase.name}.${experiment}`)
    console.log("")
    const censorJson = writeCensorJsonFile(testCase)
    const reportJson = tempFile("json")
    exec(`./miniooni -n --censor ${censorJson} -o ${reportJson} -i ${testCase.input} ${experiment}`)
    console.log("")
    const report = readReportFile(reportJson)
    const analysisResult = checker(experiment, report)
    console.log("")
    console.log("")
    switch (analysisResult) {
        case true:
        case false:
            return analysisResult
        default:
            console.log("the analysis function returned neither true nor false")
            process.exit(1)
    }
}

// runTestCase runs the given test case.
const runTestCase = (testCase) => {
    console.log("")
    console.log(`# running: ${testCase.name}`)
    let result = true
    for (const [name, checker] of Object.entries(testCase.experiments)) {
        result = result && runExperiment(testCase, name, checker)
    }
    return result
}

// checkForDuplicateTestCases ensures there are no duplicate names.
const checkForDuplicateTestCases = () => {
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
const runAllTestCases = () => {
    let result = true
    for (let i = 0; i < testCases.length; i++) {
        result = result && runTestCase(testCases[i])
    }
    return result
}

// makeTestCasesMap creates a map from the test case name
// to the test case definition.
const makeTestCasesMap = () => {
    var map = {}
    for (let i = 0; i < testCases.length; i++) {
        const testCase = testCases[i]
        map[testCase.name] = testCase
    }
    return map
}

// recompileMiniooni recompiles miniooni if needed.
const recompileMiniooni = () => {
    exec("go build -v ./internal/cmd/miniooni")
}

// commandRun implements the run command.
const commandRun = (args) => {
    recompileMiniooni()
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
const commandList = () => {
    for (let i = 0; i < testCases.length; i++) {
        const testCase = testCases[i]
        console.log(`${testCase.name}:`)
        console.log(`\t${testCase.description}`)
    }
}

// main is the main function.
const main = () => {
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
