// This script performs QA checks.

import { runTestCase } from "./lib/runner.mjs"
import { testCases as webTestCases } from "./lib/web.mjs"

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
const testCases = [
    ...webTestCases,
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
        console.log(`${testCase.name}: ${testCase.description}`)
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
