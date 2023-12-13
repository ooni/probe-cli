// Package kvstore2dir migrates $OONI_HOME/kvstore2 to $OONI_HOME/engine. This ensures
// that miniooni and ooniprobe use the same directory name for the engine state.
package kvstore2dir

import (
	"os"
	"path/filepath"
)

type statBuf interface {
	IsDir() bool
}

func simplifiedStat(path string) (statBuf, error) {
	return os.Stat(path)
}

var (
	osStat   = simplifiedStat
	osRename = os.Rename
)

// Move moves $OONI_HOME/kvstore2 to $OONI_HOME/engine, if possible.
func Move(dir string) error {
	kvstore2dir := filepath.Join(dir, "kvstore2")
	if stat, err := osStat(kvstore2dir); err != nil || !stat.IsDir() {
		return nil
	}
	enginedir := filepath.Join(dir, "engine")
	if _, err := osStat(enginedir); err == nil {
		return nil
	}
	return osRename(kvstore2dir, enginedir)
}
