package main

//
// Generates the macOS workflow.
//

import (
	"io"

	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

func buildAndPublishCLIMacOS(w io.Writer, job *Job) {
	runtimex.Assert(len(job.ArchsMatrix) <= 0, "expected no architecture matrix")

	buildJob := "build_darwin_cli"
	artifacts := []string{
		"./CLI/ooniprobe-darwin-amd64",
		"./CLI/ooniprobe-darwin-arm64",
		"./CLI/miniooni-darwin-amd64",
		"./CLI/miniooni-darwin-arm64",
	}
	testJob := "test_darwin_cli"
	publishJob := "publish_darwin_cli"

	newJob(w, buildJob, runsOnMacOS, noDependencies, noPermissions)
	newStepCheckout(w)
	newStepSetupGo(w, "macos")
	newStepSetupPsiphon(w)
	newStepMake(w, "CLI/darwin")
	newStepUploadArtifacts(w, artifacts)

	newJob(w, testJob, runsOnMacOS, buildJob, noPermissions)
	newStepDownloadArtifacts(w, []string{"ooniprobe-darwin-amd64"})
	newStepRunOONIProbeIntegrationTests(w, "darwin", "amd64", "")

	newJob(w, publishJob, runsOnMacOS, testJob, contentsWritePermissions)
	newStepDownloadArtifacts(w, artifacts)
	newStepGHPublish(w, artifacts)
}
