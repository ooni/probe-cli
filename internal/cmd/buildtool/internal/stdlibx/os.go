package stdlibx

//
// Wrappers for os
//

import (
	"io/fs"
	"os"
)

// RegularFileExists implements Stdlib.
func (sp *stdlib) RegularFileExists(filename string) bool {
	finfo, err := os.Stat(filename)
	if err != nil {
		return false
	}
	return finfo.Mode().IsRegular()
}

// MustReadFileFirstLine implements Stdlib.
func (sp *stdlib) MustReadFileFirstLine(filename string) string {
	data, err := os.ReadFile(filename)
	sp.ExitOnError(err, "os.ReadFile")
	return mustReadFirstLine(data)
}

// CopyFile implements Stdlib.
func (sp *stdlib) CopyFile(source, dest string) error {
	data, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	return os.WriteFile(dest, data, 0600)
}

// MustWriteFile implements Stdlib.
func (sp *stdlib) MustWriteFile(filename string, data []byte, perms fs.FileMode) {
	err := os.WriteFile(filename, data, perms)
	sp.ExitOnError(err, "os.WriteFile")
}
