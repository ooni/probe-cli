package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ooni/probe-cli/v3/internal/randx"
	"golang.org/x/sys/execabs"
)

// DockerImage is a docker image we can use for testing.
type DockerImage struct {
	name string
	d    *TempDir
}

// NewDockerImage builds the container image.
func NewDockerImage(config *Config, shell Shell) *DockerImage {
	dir := MkdirTemp(".", "jafar2-docker-build")
	filename := filepath.Join(dir.name, "Dockerfile")
	fp := CreateFile(filename)
	fp.WriteString("FROM alpine:3.14.1\n")
	fp.WriteString("RUN apk add iproute2\n")
	fp.MustClose()
	const imageName = "jafar2-alpine"
	cmd := NewCommandWithStdio("docker", "build", "-t", imageName, dir.Path())
	shell.MustRun(cmd)
	return &DockerImage{name: imageName, d: dir}
}

// Name returns the image name.
func (di *DockerImage) Name() string {
	return di.name
}

// Cleanup cleanups the temporary directory used to build the image.
func (di *DockerImage) Cleanup() {
	di.d.Cleanup()
}

// DockerNetwork is a docker network we can use.
type DockerNetwork struct {
	bridge string
	name   string
	shell  Shell
}

// NewDockerNetwork creates a new DockerNetwork instance.
func NewDockerNetwork(config *Config, shell Shell) *DockerNetwork {
	const networkSize = 16
	name := "jafar2-" + randx.Letters(networkSize)
	cmd := execabs.Command("docker", "network", "create", "-d", "bridge", name)
	bridge := "br-" + string(shell.MustCaptureOutput(cmd)[:12])
	return &DockerNetwork{bridge: bridge, name: name, shell: shell}
}

// Bridge returns the name of the bridge used by this network.
func (dn *DockerNetwork) Bridge() string {
	return dn.bridge
}

// Name returns the network name.
func (dn *DockerNetwork) Name() string {
	return dn.name
}

// Cleanup removes this docker network.
func (dn *DockerNetwork) Cleanup() {
	cmd := NewCommandWithStdio("docker", "network", "rm", dn.name)
	dn.shell.MustRun(cmd)
}

// DockerRun runs docker.
//
// Arguments:
//
// - config is the current config;
//
// - sh is the shell to use;
//
// - dev is the bridge device we created;
//
// - network is the docker network name;
//
// - container is the container name;
//
// - trampoline is the docker trampoline script.
func DockerRun(config *Config, sh Shell, dev string,
	network string, container string, trampoline string) {
	if config.Download != nil {
		dockerRunTcQdiscNetem(config.Download, sh, dev)
		dockerRunTcQdiscTBF(config.Download, sh, dev)
	}
	cwd, err := os.Getwd()
	FatalOnError(err, "cannot get current working directory")
	cmd := NewCommandWithStdio(
		"docker",
		"run",
		"--network",
		network,
		"--cap-add",
		"NET_ADMIN",
		"-v",
		fmt.Sprintf("%s:/ooni", cwd),
		"-w",
		"/ooni",
		"-t",
		container,
		trampoline,
	)
	sh.MustRun(cmd)
}

func dockerRunTcQdiscNetem(config *PathConstraints, sh Shell, dev string) {
	if config.Netem == "" {
		return
	}
	args := fmt.Sprintf("tc qdisc add dev %s root handle 1: netem", dev)
	arguments := strings.Split(args, " ")
	arguments = append(arguments, SplitShellArgs(config.Netem)...)
	cmd := NewCommandWithStdio("sudo", arguments...)
	sh.MustRun(cmd)
}

func dockerRunTcQdiscTBF(config *PathConstraints, sh Shell, dev string) {
	if config.TBF == "" {
		return
	}
	parent := "root"
	if config.Netem != "" {
		parent = "parent 1:"
	}
	args := fmt.Sprintf("tc qdisc add dev %s %s handle 2: tbf", dev, parent)
	arguments := strings.Split(args, " ")
	arguments = append(arguments, SplitShellArgs(config.TBF)...)
	cmd := NewCommandWithStdio("sudo", arguments...)
	sh.MustRun(cmd)
}
