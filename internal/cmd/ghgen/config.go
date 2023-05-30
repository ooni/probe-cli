package main

//
// Configuration with which we will run
//

import "io"

// Job is a job to run.
type Job struct {
	// Action is the job name
	Action func(w io.Writer, job *Job)

	// ArchsMatrix contains the architectures to iterate over
	ArchsMatrix []string
}

// Config contains the configuration.
var Config = map[string][]Job{
	"android": {{
		Action: buildAndPublishAndroid,
		ArchsMatrix: []string{
			"386",
			"amd64",
			"arm",
			"arm64",
		},
	}},
	"ios": {{
		Action:      buildAndPublishMobileIOS,
		ArchsMatrix: []string{},
	}},
	"linux": {{
		Action: buildAndPublishCLILinux,
		ArchsMatrix: []string{
			"386",
			"amd64",
			"armv6",
			"armv7",
			"arm64",
		},
	}},
	"macos": {{
		Action:      buildAndPublishCLIMacOS,
		ArchsMatrix: []string{},
	}},
	"windows": {{
		Action:      buildAndPublishCLIWindows,
		ArchsMatrix: []string{},
	}},
}

const (
	// runOnUbuntu is the Ubuntu system where to run.
	runsOnUbuntu = "ubuntu-22.04"

	// runsOnMacOS is the macOS system where to run.
	runsOnMacOS = "macos-12"

	// runsOnWindows is the windows system where to run.
	runsOnWindows = "windows-2022"
)

// noPermission indicates a job does not require permissions.
var noPermissions map[string]string

// contentsWritePermissions indicates the job needs the `contents: write` permission.
var contentsWritePermissions = map[string]string{
	"contents": "write",
}

// noDependencies indicates a job does not require dependencies.
var noDependencies string
