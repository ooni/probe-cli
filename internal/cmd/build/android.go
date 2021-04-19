package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine/runtimex"
	"github.com/ooni/probe-cli/v3/internal/engine/shellx"
)

// AndroidCmd is the "android" command.
type AndroidCmd struct {
	// mobileCmd indicates that AndroidCmd is a mobile command.
	mobileCmd

	// Bundle indicates whether to create bundle.jar.
	Bundle bool `help:"Create bundle.jar for Maven Central (requires --sign)."`

	// EmbedPsiphonConfig tells the build procedure to embed
	// try to embed a suitable psiphon configuration.
	EmbedPsiphonConfig bool `help:"Try to embed a suitable psiphon configuration"`

	// Sign indicates the PGP identity that should sign the generated files.
	Sign string `help:"PGP identity that should sign the generated files." placeholder:"EMAIL"`
}

// androidBuildSpec contains info about a build.
type androidBuildSpec struct {
	// destDir is the directory where to store generated files.
	destDir string

	// version is the version number.
	version string
}

// Run runs the "android command"
func (cmd *AndroidCmd) Run(flags *GlobalFlags) error {
	if cmd.Bundle && cmd.Sign == "" {
		return errors.New("you need to specify --sign=EMAIL with --bundle")
	}
	androidHome := os.Getenv("ANDROID_HOME")
	if androidHome == "" {
		return errors.New("you need to set ANDROID_HOME")
	}
	androidNdkHome := os.Getenv("ANDROID_NDK_HOME")
	if androidNdkHome == "" {
		return errors.New("you need to set ANDROID_NDK_HOME")
	}
	spec := &androidBuildSpec{
		destDir: path.Join("MOBILE", "android"),
		version: time.Now().Format("2006.01.02-150405"),
	}
	_ = os.RemoveAll(spec.destDir)
	must(os.MkdirAll(spec.destDir, 0755))
	cmd.mobileCmd.gomobileInit(flags)
	cmd.gomobileBind(flags, spec)
	cmd.writePOM(spec)
	cmd.maybeSign(spec)
	cmd.maybeCreateBundle(spec)
	return nil
}

// pomTemplate contains the POM file template.
const pomTemplate string = `<?xml version="1.0" encoding="UTF-8"?>
<project xsi:schemaLocation="http://maven.apache.org/POM/4.0.0 http://maven.apache.org/xsd/maven-4.0.0.xsd" xmlns="http://maven.apache.org/POM/4.0.0"
    xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <modelVersion>4.0.0</modelVersion>

  <groupId>org.ooni</groupId>
  <artifactId>oonimkall</artifactId>
  <version>@VERSION@</version>
  <packaging>aar</packaging>

  <name>oonimkall</name>
  <description>OONI Probe Library for Android</description>
  <url>https://github.com/ooni/probe-cli</url>

  <licenses>
    <license>
      <name>The 3-Clause BSD License</name>
      <url>https://opensource.org/licenses/BSD-3-Clause</url>
      <distribution>repo</distribution>
    </license>
  </licenses>

  <scm>
    <url>https://github.com/ooni/probe-engine</url>
    <connection>https://github.com/ooni/probe-engine.git</connection>
  </scm>

  <developers>
    <developer>
      <name>Simone Basso</name>
      <email>simone@openobservatory.org</email>
      <roles>
        <role>Core developer</role>
      </roles>
      <timezone>Europe/Rome</timezone>
    </developer>
  </developers>

</project>
`

// writePom writes the POM file for the given spec.
func (cmd *AndroidCmd) writePOM(spec *androidBuildSpec) {
	data := strings.ReplaceAll(pomTemplate, "@VERSION@", spec.version)
	filepath := path.Join(spec.destDir, fmt.Sprintf("oonimkall-%s.pom", spec.version))
	must(ioutil.WriteFile(filepath, []byte(data), 0644))
}

// maybeSign signs the generated files if needed
func (cmd *AndroidCmd) maybeSign(spec *androidBuildSpec) {
	if cmd.Sign == "" {
		return
	}
	matches, err := filepath.Glob(path.Join(spec.destDir, "oonimkall-*"))
	runtimex.PanicOnError(err, "filepath.Glob failed")
	for _, m := range matches {
		must(shellx.Run("gpg", "-ab", "-u", cmd.Sign, m))
	}
}

// gomobileBind runs gomobile bind
func (cmd *AndroidCmd) gomobileBind(flags *GlobalFlags, spec *androidBuildSpec) {
	var args []string
	args = append(args, "bind")
	args = append(args, "-target")
	args = append(args, "android")
	args = append(args, "-o")
	aar := path.Join(spec.destDir, fmt.Sprintf("oonimkall-%s.aar", spec.version))
	args = append(args, aar)
	args = append(args, "-ldflags")
	args = append(args, "-s -w")
	if cmd.EmbedPsiphonConfig {
		args = append(args, "-tags")
		args = append(args, "ooni_psiphon_config")
	}
	if flags.Verbose {
		args = append(args, "-v")
	}
	args = append(args, "./pkg/oonimkall")
	must(shellx.Run("gomobile", args...))
}

// maybeCreateBundle creates bundle.jar if needed
func (cmd *AndroidCmd) maybeCreateBundle(spec *androidBuildSpec) {
	if !cmd.Bundle {
		return
	}
	cwd := mustString(os.Getwd())
	must(os.Chdir(spec.destDir))
	var args []string
	args = append(args, "-cf")
	args = append(args, "bundle.jar")
	matches, err := filepath.Glob(path.Join(".", "oonimkall-*"))
	runtimex.PanicOnError(err, "filepath.Glob failed")
	args = append(args, matches...)
	must(shellx.Run("jar", args...))
	must(os.Chdir(cwd))
}
