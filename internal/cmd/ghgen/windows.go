package main

//
// Generates the Windows workflow.
//

import (
	"io"

	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

func buildAndPublishCLIWindows(w io.Writer, job *Job) {
	runtimex.Assert(len(job.ArchsMatrix) <= 0, "expected no architecture matrix")

	buildJob := "build_windows_cli"
	artifacts := []string{
		"./CLI/ooniprobe-windows-386.exe",
		"./CLI/ooniprobe-windows-amd64.exe",
		"./CLI/miniooni-windows-386.exe",
		"./CLI/miniooni-windows-amd64.exe",
	}
	testJob := "test_windows_cli"
	publishJob := "publish_windows_cli"

	newJob(w, buildJob, runsOnUbuntu, noDependencies, noPermissions)
	newStepCheckout(w)
	newStepSetupGo(w, "windows")
	newStepInstallMingwW64(w)
	newStepSetupPsiphon(w)
	newStepMake(w, "EXPECTED_MINGW_W64_VERSION=\"9.3-win32\" CLI/windows")

	newStepUploadArtifacts(w, artifacts)

	newJob(w, testJob, runsOnWindows, buildJob, noPermissions)
	newStepCheckout(w)
	newStepDownloadArtifacts(w, []string{"ooniprobe-windows-amd64.exe"})
	newStepRunOONIProbeIntegrationTests(w, "windows", "amd64", ".exe")

	newJob(w, publishJob, runsOnUbuntu, testJob, contentsWritePermissions)
	newStepCheckout(w)
	newStepDownloadArtifacts(w, artifacts)
	newStepGHPublish(w, artifacts)
}
