package main

//
// Psiphon specific stuff
//

import (
	"path"
	"path/filepath"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/fsx"
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
		log.Infof("psiphon files already present on the filesystem")
		return
	}
	log.Infof("trying to copy psiphon config files")
	privateRepoDir := filepath.Join("MONOREPO", "repo", "probe-private")
	err := psiphonAttemptToCopyConfig(privateRepoDir)
	if err != nil {
		log.Infof("trying to clone github.com/ooni/probe-private")
		err := shellx.Run(log.Log, "git", "clone", "git@github.com:ooni/probe-private", privateRepoDir)
		if err != nil {
			log.Warnf("it seems we cannot clone ooni/probe-private")
			return
		}
		if err := psiphonAttemptToCopyConfig(privateRepoDir); err != nil {
			log.Warnf("it seems we cannot copy psiphon config")
			return
		}
	}
	log.Infof("psiphon config files copied successfully")
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
