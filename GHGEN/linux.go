package main

//
// Generates Linux workflow.
//

import (
	"fmt"
	"io"

	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

func buildAndPublishCLILinux(w io.Writer, job *Job) {
	runtimex.Assert(len(job.ArchsMatrix) > 0, "expected architecture matrix")

	for _, arch := range job.ArchsMatrix {
		buildJob := fmt.Sprintf("build_linux_cli_%s", arch)
		artifacts := []string{
			fmt.Sprintf("./CLI/ooniprobe-linux-%s", arch),
			fmt.Sprintf("./CLI/miniooni-linux-%s", arch),
		}
		testJob := fmt.Sprintf("test_linux_cli_%s", arch)
		publishJob := fmt.Sprintf("publish_linux_cli_%s", arch)

		newJob(w, buildJob, runsOnUbuntu, noDependencies, noPermissions)
		newStepCheckout(w)
		switch arch {
		case "386", "amd64":
			// nothing
		default:
			newSetupInstallQemuUserStatic(w)
		}
		newStepSetupPsiphon(w)
		newStepSetupLinuxDockerGoCache(w, arch)
		newStepMake(w, fmt.Sprintf("CLI/linux-static-%s", arch))
		newStepUploadArtifacts(w, artifacts)

		// We only run integration tests for amd64
		switch arch {
		case "amd64":
			newJob(w, testJob, runsOnUbuntu, buildJob, noPermissions)
			newStepCheckout(w)
			newStepDownloadArtifacts(w, artifacts)
			newStepSetupGo(w, fmt.Sprintf("linux-%s", arch))
			newStepInstallTor(w)
			newStepRunOONIProbeIntegrationTests(w, "linux", arch, "")
			newStepRunMiniooniIntegrationTests(w, arch, "")
			newJob(w, publishJob, runsOnUbuntu, testJob, contentsWritePermissions)
		default:
			newJob(w, publishJob, runsOnUbuntu, buildJob, contentsWritePermissions)
		}

		newStepCheckout(w)
		newStepDownloadArtifacts(w, artifacts)
		newStepGHPublish(w, artifacts)
	}
}
