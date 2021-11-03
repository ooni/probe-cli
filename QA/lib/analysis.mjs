// This file contains code for analysing results.

import { type } from "os"

// checkAnnotations checks whether we have annotations.
function checkAnnotations(report) {
    let result = true

    const isObject = typeof (report.annotations) === "object"
    console.log(`checking whether annotations is an object... ${isObject}`)
    result = result && isObject

    const hasArchitecture = typeof (report.annotations.architecture) === "string"
    console.log(`checking whether annotations contains architecture... ${hasArchitecture}`)
    result = result && hasArchitecture

    const hasEngineName = typeof (report.annotations.engine_name) === "string"
    console.log(`checking whether annotations contains engine_name... ${hasEngineName}`)
    result = result && hasEngineName

    const hasEngineVersion = typeof (report.annotations.engine_version) === "string"
    console.log(`checking whether annotations contains engine_version... ${hasEngineVersion}`)
    result = result && hasEngineVersion

    const hasPlatform = typeof (report.annotations.platform) === "string"
    console.log(`checking whether annotations contains platform... ${hasPlatform}`)
    result = result && hasPlatform

    return result
}

// checkDataFormatVersion checks whether we have data_format_version.
function checkDataFormatVersion(report) {
    const result = report.data_format_version === "0.2.0"
    console.log(`checking whether we have the right data format version... ${result}`)
    return result
}

// checkExtensions ensures that extensions exists and has the right type.
function checkExtensions(report) {
    const result = typeof (report.extensions) === "object"
    console.log(`checking whether report.extensions is an object... ${result}`)
    // Quirk: some experiments (e.g. web_connectivity) don't include
    // an extensions object, and we don't want to fail here.
    return true
}

// checkInput ensures that the input is correct.
function checkInput(testCase, report) {
    const result = testCase.input === report.input
    console.log(`checking whether input is correct... ${result}`)
    return result
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

// checkStartTime ensures that a given start time is in the correct format.
function checkStartTime(report, key) {
    const value = report[key]
    let result = typeof (value) === "string"
    console.log(`checking whether ${key} is a string... ${result}`)
    result = result && value.match(/^[0-9]{4}-[0-9]{2}-[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2}$/) !== null
    console.log(`checking whether ${key} matches the date regexp... ${result}`)
    return result
}

// checkASN ensures that an ASN field is correct.
function checkASN(report, key) {
    const value = report[key]
    let result = typeof (value) === "string"
    console.log(`checking whether ${key} is a string... ${result}`)
    result = result && value.match(/^AS[0-9]+$/) !== null
    console.log(`checking whether ${key} matches the ASN regexp... ${result}`)
    return result
}

// checkProbeCC ensures that the probe_cc field is correct.
function checkProbeCC(report) {
    const value = report["probe_cc"]
    let result = typeof (value) === "string"
    console.log(`checking whether probe_cc is a string... ${result}`)
    result = result && value.match(/^[A-Z]{2}$/) !== null
    console.log(`checking whether probe_cc matches the CC regexp... ${result}`)
    return result

}

// checkProbeIP ensures that the probe_ip field is correct.
function checkProbeIP(report) {
    const result = report.probe_ip === "127.0.0.1"
    console.log(`checking whether probe_ip is correct... ${result}`)
    return result
}

// checkReportID ensures that the report_id field is correct.
function checkReportID(report) {
    const result = report.report_id === "" // note: we don't submit
    console.log(`checking whether report_id is correct... ${result}`)
    return result
}

// checkString ensures that a field is a string.
function checkString(report, key) {
    const value = report[key]
    const result = typeof(value) === "string"
    console.log(`checking whether ${key} is a string... ${result}`)
    return result
}

// checkNetworkName ensures that an xxx_network_name field is correct.
function checkNetworkName(report, key) {
    return checkString(report, key)
}

// checkResolverIP ensures that the resolver_ip field is correct.
function checkResolverIP(report) {
    return checkString(report, "resolver_ip")
}

// checkSoftwareName ensures that the software_name field is correct.
function checkSoftwareName(report) {
    return checkString(report, "software_name")
}

// checkSoftwareVersion ensures that the software_version field is correct.
function checkSoftwareVersion(report) {
    return checkString(report, "software_version")
}

// checkTestRuntime ensures that the test_runtime field is correct.
function checkTestRuntime(report) {
    const result = typeof(report.test_runtime) === "number"
    console.log(`checking whether test_runtime is a number... ${result}`)
    return result
}

// checkTestVersion ensures that the test_version field is correct.
function checkTestVersion(report) {
    return checkString(report, "test_version")
}

// checkTestKeys ensures that test_keys is an object.
function checkTestKeys(report) {
    const result = typeof(report.test_keys) === "object"
    console.log(`checking whether test_keys is an object... ${result}`)
    return result
}

// checkMeasurement is a function that invokes a standard set of checks
// to validate the top-level test keys of a result.
//
// This function helps to implement per-experiment checkers.
//
// Arguments:
//
// - testCase is the current test case
//
// - name is the name of the current experiment
//
// - report is the JSON measurement
//
// - extraChecks is the optional callback to perform extra checks
// that takes in input the testCase, the name, and the report.
export function checkMeasurement(testCase, name, report, extraChecks) {
    let result = true
    result = result && checkAnnotations(report)
    result = result && checkDataFormatVersion(report)
    result = result && checkExtensions(report)
    result = result && checkInput(testCase, report)
    result = result && checkStartTime(report, "measurement_start_time")
    result = result && checkASN(report, "probe_asn")
    result = result && checkProbeCC(report)
    result = result && checkProbeIP(report)
    result = result && checkNetworkName(report, "probe_network_name")
    result = result && checkReportID(report)
    result = result && checkASN(report, "resolver_asn")
    result = result && checkResolverIP(report)
    result = result && checkNetworkName(report, "resolver_network_name")
    result = result && checkSoftwareName(report)
    result = result && checkSoftwareVersion(report)
    result = result && checkExperimentName(name, report)
    result = result && checkTestRuntime(report)
    result = result && checkStartTime(report, "test_start_time")
    result = result && checkTestVersion(report)
    result = result && checkTestKeys(report)
    if (typeof extraChecks === "function") {
        result = result && extraChecks(report.test_keys)
    }
    return result
}
