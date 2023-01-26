package main

//
// Generates Android workflow.
//

import (
	"fmt"
	"io"

	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

func buildAndPublishAndroid(w io.Writer, job *Job) {
	runtimex.Assert(len(job.ArchsMatrix) > 0, "expected architecture matrix")

	buildJob := "build_android"
	artifacts := []string{
		"./MOBILE/android/oonimkall.aar",
		"./MOBILE/android/oonimkall-sources.jar",
		"./MOBILE/android/oonimkall.pom",
	}
	for _, arch := range job.ArchsMatrix {
		artifacts = append(artifacts, fmt.Sprintf("./CLI/miniooni-android-%s", arch))
		artifacts = append(artifacts, fmt.Sprintf("./CLI/ooniprobe-android-%s", arch))
	}

	publishJob := "publish_android"

	newJob(w, buildJob, runsOnUbuntu, noDependencies, noPermissions)
	newStepCheckout(w)
	newStepSetupGo(w, "android")
	newStepSetupPsiphon(w)
	newStepMake(w, "MOBILE/cli")
	newStepMake(w, "MOBILE/android")
	newStepUploadArtifacts(w, artifacts)

	newJob(w, publishJob, runsOnUbuntu, buildJob, contentsWritePermissions)
	newStepCheckout(w)
	newStepDownloadArtifacts(w, artifacts)
	newStepGHPublish(w, artifacts)
}
