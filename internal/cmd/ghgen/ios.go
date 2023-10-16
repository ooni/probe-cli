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
	iosNewStepBrewInstall(w)
	newStepMake(w, "EXPECTED_XCODE_VERSION=14.2 ios")
	newStepUploadArtifacts(w, artifacts)

	newJob(w, publishJob, runsOnUbuntu, buildJob, contentsWritePermissions)
	newStepCheckout(w)
	newStepDownloadArtifacts(w, artifacts)
	newStepGHPublish(w, artifacts)
}

func iosNewStepBrewInstall(w io.Writer) {
	mustFprintf(w, "      # ./internal/cmd/buildtool needs coreutils for sha256 plus GNU build tools\n")
	mustFprintf(w, "      - run: brew install autoconf automake coreutils libtool\n")
	mustFprintf(w, "\n")
}
