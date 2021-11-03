// This file contains code for running test cases.

import child_process from "child_process"
import crypto from "crypto"
import fs from "fs"
import path from "path"

// tempFile returns the name of a temporary file. This function is
// not as secure as using mktemp but it does not matter in this
// context. We just need to create file names in the local directory
// with enough entropy that every run has a different name.
//
// See https://stackoverflow.com/questions/7055061 for an insightful
// discussion on how one should or should not create temp files.
function tempFile(suffix) {
    return path.join(`tmp-${crypto.randomBytes(16).toString('hex')}.${suffix}`)
}

// exec executes a command. This function throws on failure.
function exec(command) {
    console.log(`+ ${command}`)
    child_process.execSync(command)
}

// writeCensorJsonFile writes a censor.json file using a file name
// containing random characters and returns the file name.
function writeCensorJsonFile(testCase) {
    const fileName = tempFile("json")
    fs.writeFileSync(fileName, JSON.stringify(testCase.blocking))
    return fileName
}

// readReportFile reads and parses the report file, thus returning
// the JSON object contained inside the report file.
function readReportFile(reportFile) {
    const data = fs.readFileSync(reportFile, { "encoding": "utf-8" })
    return JSON.parse(data)
}

// runExperiment runs the given test case with the given experiment.
function runExperiment(testCase, experiment, checker) {
    console.log(`## running: ${testCase.name}.${experiment}`)
    console.log("")
    const censorJson = writeCensorJsonFile(testCase)
    const reportJson = tempFile("json")
    exec(`./miniooni -n --censor ${censorJson} -o ${reportJson} -i ${testCase.input} ${experiment}`)
    console.log("")
    const report = readReportFile(reportJson)
    const analysisResult = checker(testCase, experiment, report)
    console.log("")
    console.log("")
    if (analysisResult !== true && analysisResult !== false) {
       console.log("the analysis function returned neither true nor false")
       process.exit(1)
    }
    return analysisResult
}

// recompileMiniooni recompiles miniooni if needed.
function recompileMiniooni() {
    exec("go build -v ./internal/cmd/miniooni")
}

// runTestCase runs the given test case.
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
// - experiments (object): the keys are names of nettests
// to run and the values are functions taking three arguments:
//
// - the test case structure
//
// - the name of the current experiment
//
// - the JSON report
export function runTestCase(testCase) {
    recompileMiniooni()
    console.log("")
    console.log(`# running: ${testCase.name}`)
    let result = true
    for (const [name, checker] of Object.entries(testCase.experiments)) {
        result = result && runExperiment(testCase, name, checker)
    }
    return result
}
