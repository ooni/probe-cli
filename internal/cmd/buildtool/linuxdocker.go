package main

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"time"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/cmd/buildtool/internal/buildtoolmodel"
	"github.com/ooni/probe-cli/v3/internal/must"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/spf13/cobra"
)

// linuxDockerSubcommand returns the linuxDocker sucommand.
func linuxDockerSubcommand() *cobra.Command {
	return &cobra.Command{
		Use:   "docker {386|amd64|armv6|armv7|arm64}",
		Short: "Builds ooniprobe and miniooni with static linking using docker",
		Run: func(cmd *cobra.Command, args []string) {
			linuxDockerBuildAll(&buildDeps{}, args[0])
		},
		Args: cobra.ExactArgs(1),
	}
}

// main is the main function of the linuxDocker subcommand.
func linuxDockerBuildAll(deps buildtoolmodel.Dependencies, ooniArch string) {
	defer log.Infof("done")
	deps.PsiphonMaybeCopyConfigFiles()

	golangVersion := string(must.FirstLineBytes(deps.LinuxReadGOVERSION("GOVERSION")))
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

	// Implementation note: we must run docker as the user that invokes
	// it for actions/cache@v3 to be able to cache OOGOCACHEDIR. This
	// constraint forces us to run all privileged operations early
	// using a Dockerfile, so the build proper runs as $(id -u):$(id -g).
	log.Infof("writing CLI/Dockerfile")
	linuxDockerWriteDockerfile(deps, dockerArch, golangDockerImage, user.Uid)

	image := fmt.Sprintf("oobuild-%s-%s", ooniArch, time.Now().Format("20060102"))

	log.Infof("pull and build the correct docker image")
	must.Run(log.Log, "docker", "pull", "--platform", "linux/"+dockerArch, golangDockerImage)
	must.Run(log.Log, "docker", "build", "--platform", "linux/"+dockerArch, "-t", image, "CLI")

	log.Infof("run the build inside docker")
	curdir := runtimex.Try1(os.Getwd())

	must.Run(
		log.Log, "docker", "run",
		"--platform", "linux/"+dockerArch,
		"--user", user.Uid,
		"-v", curdir+":/ooni",
		"-w", "/ooni",
		image,
		"go", "run", "./internal/cmd/buildtool", "linux", "static", "--goarm", goarm,
	)
}

// linuxDockerWwriteDockerfile writes the CLI/Dockerfile file.
func linuxDockerWriteDockerfile(deps buildtoolmodel.Dependencies, dockerArch, golangDockerImage, uid string) {
	content := []byte(fmt.Sprintf(`
		FROM --platform=linux/%s %s
		RUN apk update
		RUN apk upgrade
		RUN apk add --no-progress gcc git linux-headers musl-dev
		RUN adduser -D -h /home/oobuild -G nobody -u %s oobuild
		ENV HOME=/home/oobuild`, dockerArch, golangDockerImage, uid,
	))
	deps.LinuxWriteDockerfile(filepath.Join("CLI", "Dockerfile"), content, 0600)
}
