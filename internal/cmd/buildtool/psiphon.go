package main

//
// Psiphon specific stuff
//

import (
	"os"
	"path"
	"path/filepath"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/fsx"
	"github.com/ooni/probe-cli/v3/internal/must"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/shellx"
)

// psiphonConfigJSONAge is the psiphon-config.json.age full file path.
var psiphonConfigJSONAge = filepath.Join("internal", "engine", "psiphon-config.json.age")

// psiphonConfigKey is the psiphon-config.key full file path.
var psiphonConfigKey = filepath.Join("internal", "engine", "psiphon-config.key")

// psiphonFilesExist returns true when psiphon files are on the filesystem.
func psiphonFilesExist() bool {
	return fsx.RegularFileExists(psiphonConfigJSONAge) && fsx.RegularFileExists(psiphonConfigKey)
}

// psiphonMaybeCopyConfigFiles copies the psiphon config if possible.
func psiphonMaybeCopyConfigFiles() {
	if psiphonFilesExist() {
		must.Fprintf(os.Stderr, "# psiphon files already present on the filesystem\n")
		return
	}
	must.Fprintf(os.Stderr, "# trying to copy psiphon config files\n")
	privateRepoDir := filepath.Join("MONOREPO", "repo", "probe-private")
	err := psiphonAttemptToCopyConfig(privateRepoDir)
	if err != nil {
		must.Fprintf(os.Stderr, "# trying to clone github.com/ooni/probe-private\n")
		must.Run(log.Log, "git", "clone", "git@github.com:ooni/probe-private", privateRepoDir)
		runtimex.Try0(psiphonAttemptToCopyConfig(privateRepoDir))
	}
	must.Fprintf(os.Stderr, "# psiphon config files copied successfully\n")
	must.Fprintf(os.Stderr, "\n")
}

// psiphonAttemptToCopyConfig attempts to copy the config from the monorepo.
func psiphonAttemptToCopyConfig(prefix string) error {
	candidates := []string{
		psiphonConfigJSONAge,
		psiphonConfigKey,
	}
	for _, candidate := range candidates {
		source := filepath.Join(prefix, path.Base(candidate))
		dest := candidate
		if err := shellx.CopyFile(source, dest, 0600); err != nil {
			return err
		}
	}
	return nil
}
