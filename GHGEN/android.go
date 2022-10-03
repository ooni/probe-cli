package main

//
// Generates Android workflow.
//

import (
	"fmt"
	"io"

	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

func buildAndPublishMobileAndroid(w io.Writer, job *Job) {
	runtimex.Assert(len(job.ArchsMatrix) <= 0, "expected no architecture matrix")

	buildJob := "build_android_mobile"
	artifacts := []string{
		"./MOBILE/android/oonimkall.aar",
		"./MOBILE/android/oonimkall-sources.jar",
		"./MOBILE/android/oonimkall.pom",
	}
	publishJob := "publish_android_mobile"

	newJob(w, buildJob, runsOnUbuntu, noDependencies, noPermissions)
	newStepCheckout(w)
	newStepSetupGo(w, "android-oonimkall")
	newStepSetupPsiphon(w)
	newStepMake(w, "MOBILE/android")
	newStepUploadArtifacts(w, artifacts)

	newJob(w, publishJob, runsOnUbuntu, buildJob, contentsWritePermissions)
	newStepCheckout(w)
	newStepDownloadArtifacts(w, artifacts)
	newStepGHPublish(w, artifacts)
}

func buildAndPublishCLIAndroid(w io.Writer, job *Job) {
	runtimex.Assert(len(job.ArchsMatrix) > 0, "expected architecture matrix")

	for _, arch := range job.ArchsMatrix {
		buildJob := fmt.Sprintf("build_android_cli_%s", arch)
		artifacts := []string{
			fmt.Sprintf("./CLI/miniooni-android-%s", arch),
			fmt.Sprintf("./CLI/ooniprobe-android-%s", arch),
		}
		publishJob := fmt.Sprintf("publish_android_cli_%s", arch)

		newJob(w, buildJob, runsOnUbuntu, noDependencies, noPermissions)
		newStepCheckout(w)
		newStepSetupGo(w, fmt.Sprintf("android-cli-%s", arch))
		newStepSetupPsiphon(w)
		newStepMake(w, fmt.Sprintf("CLI/android-%s", arch))
		newStepUploadArtifacts(w, artifacts)

		newJob(w, publishJob, runsOnUbuntu, buildJob, contentsWritePermissions)
		newStepCheckout(w)
		newStepDownloadArtifacts(w, artifacts)
		newStepGHPublish(w, artifacts)
	}
}
