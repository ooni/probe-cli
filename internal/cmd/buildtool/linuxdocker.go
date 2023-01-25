package main

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"time"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/must"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/shellx"
	"github.com/spf13/cobra"
)

// linuxDockerSubcommand returns the linuxDocker sucommand.
func linuxDockerSubcommand() *cobra.Command {
	return &cobra.Command{
		Use:   "linux-docker {386|amd64|armv6|armv7|arm64}",
		Short: "Builds ooniprobe for linux-static using docker and golang:alpine",
		Run: func(cmd *cobra.Command, args []string) {
			linuxDockerBuildAll(args[0])
		},
		Args: cobra.ExactArgs(1),
	}
}

// main is the main function of the linuxDocker subcommand.
func linuxDockerBuildAll(ooniArch string) {
	psiphonMaybeCopyConfigFiles()

	golangVersion := string(must.FirstLineBytes(must.ReadFile("GOVERSION")))
	golangDockerImage := "golang:" + golangVersion + "-alpine"

	var (
		goarm      string
		dockerArch string
	)
	switch ooniArch {
	case "armv7":
		goarm = "7"
		dockerArch = "arm/v7"
	case "armv6":
		goarm = "6"
		dockerArch = "arm/v6"
	default:
		goarm = "0"
		dockerArch = ooniArch
	}

	user := runtimex.Try1(user.Current())

	must.Fprintf(os.Stderr, "# writing CLI/Dockerfile\n")
	linuxDockerWriteDockerfile(dockerArch, golangDockerImage, user.Uid)
	must.Fprintf(os.Stderr, "\n")

	taggedImage := fmt.Sprintf(
		"oobuild-%s-%s", ooniArch, time.Now().Format("20060102"),
	)

	must.Fprintf(os.Stderr, "# pull the correct docker image\n")
	must.Run(log.Log, "docker", "pull", "--platform", "linux/"+dockerArch, golangDockerImage)
	must.Fprintf(os.Stderr, "\n")

	linuxDockerMaybeRebuildTag(dockerArch, taggedImage)

	must.Fprintf(os.Stderr, "# run the build inside docker\n")
	curdir := runtimex.Try1(os.Getwd())

	must.Run(
		log.Log, "docker", "run",
		"--platform", "linux/"+dockerArch,
		"--user", user.Uid,
		"-v", curdir+":/ooni",
		"-w", "/ooni",
		taggedImage,
		"go", "run", "./internal/cmd/buildtool", "build", "linux-static", "--goarm", goarm,
	)
}

// linuxDockerMaybeRebuildTag rebuilds the image tag if needed.
func linuxDockerMaybeRebuildTag(dockerArch, taggedImage string) {
	must.Fprintf(os.Stderr, "# see whether we need to rebuild and retag\n")
	err := shellx.RunQuiet("docker", "inspect", "--type", "image", taggedImage)
	if err != nil {
		// TODO(bassosimone): maybe we could be more precise with checking
		// this error here because this check is a bit too broad?
		must.Run(
			log.Log, "docker", "build",
			"--platform", "linux/"+dockerArch,
			"-t", taggedImage,
			"CLI",
		)
	}
	must.Fprintf(os.Stderr, "\n")
}

// linuxDockerWwriteDockerfile writes the CLI/Dockerfile file.
func linuxDockerWriteDockerfile(dockerArch, golangDockerImage, uid string) {
	content := []byte(fmt.Sprintf(`
		FROM --platform=linux/%s %s
		RUN apk update
		RUN apk upgrade
		RUN apk add --no-progress gcc git linux-headers musl-dev
		RUN adduser -D -h /home/oobuild -G nobody -u %s oobuild
		ENV HOME=/home/oobuild`, dockerArch, golangDockerImage, uid,
	))
	must.WriteFile(filepath.Join("CLI", "Dockerfile"), content, 0600)
}
