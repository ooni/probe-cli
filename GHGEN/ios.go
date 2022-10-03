package main

//
// Generates iOS workflow.
//

import (
	"io"

	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

func buildAndPublishMobileIOS(w io.Writer, job *Job) {
	runtimex.Assert(len(job.ArchsMatrix) <= 0, "expected no architecture matrix")

	buildJob := "build_ios_mobile"
	artifacts := []string{
		"./MOBILE/ios/oonimkall.xcframework.zip",
		"./MOBILE/ios/oonimkall.podspec",
	}
	publishJob := "publish_ios_mobile"

	newJob(w, buildJob, runsOnMacOS, noDependencies, noPermissions)
	newStepCheckout(w)
	newStepSetupGo(w, "ios")
	newStepSetupPsiphon(w)
	newStepMake(w, "EXPECTED_XCODE_VERSION=12.4 MOBILE/ios")
	newStepUploadArtifacts(w, artifacts)

	newJob(w, publishJob, runsOnUbuntu, buildJob, contentsWritePermissions)
	newStepCheckout(w)
	newStepDownloadArtifacts(w, artifacts)
	newStepGHPublish(w, artifacts)
}
