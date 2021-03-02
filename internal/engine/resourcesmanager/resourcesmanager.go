// Package resourcesmanager contains the resources manager.
package resourcesmanager

import (
	"compress/gzip"
	"crypto/sha256"
	"embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/ooni/probe-cli/v3/internal/engine/resources"
)

// Errors returned by this package.
var (
	ErrDestDirEmpty   = errors.New("resources: DestDir is empty")
	ErrSHA256Mismatch = errors.New("resources: sha256 mismatch")
)

// CopyWorker ensures that resources are current. You always need to set
// the DestDir attribute. All the rest is optional.
type CopyWorker struct {
	DestDir   string                                                     // mandatory
	Different func(left, right string) bool                              // optional
	Equal     func(left, right string) bool                              // optional
	MkdirAll  func(path string, perm os.FileMode) error                  // optional
	NewReader func(r io.Reader) (io.ReadCloser, error)                   // optional
	Open      func(path string) (fs.File, error)                         // optional
	ReadAll   func(r io.Reader) ([]byte, error)                          // optional
	ReadFile  func(filename string) ([]byte, error)                      // optional
	WriteFile func(filename string, data []byte, perm fs.FileMode) error // optional
}

//go:embed *.mmdb.gz
var efs embed.FS

func (cw *CopyWorker) mkdirAll(path string, perm os.FileMode) error {
	if cw.MkdirAll != nil {
		return cw.MkdirAll(path, perm)
	}
	return os.MkdirAll(path, perm)
}

// Ensure ensures that the resources on disk are current.
func (cw *CopyWorker) Ensure() error {
	if cw.DestDir == "" {
		return ErrDestDirEmpty
	}
	if err := cw.mkdirAll(cw.DestDir, 0700); err != nil {
		return err
	}
	for name, resource := range resources.All {
		if err := cw.ensureFor(name, &resource); err != nil {
			return err
		}
	}
	return nil
}

func (cw *CopyWorker) readFile(path string) ([]byte, error) {
	if cw.ReadFile != nil {
		return cw.ReadFile(path)
	}
	return ioutil.ReadFile(path)
}

func (cw *CopyWorker) equal(left, right string) bool {
	if cw.Equal != nil {
		return cw.Equal(left, right)
	}
	return left == right
}

func (cw *CopyWorker) different(left, right string) bool {
	if cw.Different != nil {
		return cw.Different(left, right)
	}
	return left != right
}

func (cw *CopyWorker) open(path string) (fs.File, error) {
	if cw.Open != nil {
		return cw.Open(path)
	}
	return efs.Open(path)
}

func (cw *CopyWorker) newReader(r io.Reader) (io.ReadCloser, error) {
	if cw.NewReader != nil {
		return cw.NewReader(r)
	}
	return gzip.NewReader(r)
}

func (cw *CopyWorker) readAll(r io.Reader) ([]byte, error) {
	if cw.ReadAll != nil {
		return cw.ReadAll(r)
	}
	return ioutil.ReadAll(r)
}

func (cw *CopyWorker) writeFile(filename string, data []byte, perm fs.FileMode) error {
	if cw.WriteFile != nil {
		return cw.WriteFile(filename, data, perm)
	}
	return ioutil.WriteFile(filename, data, perm)
}

func (cw *CopyWorker) sha256sum(data []byte) string {
	return fmt.Sprintf("%x", sha256.Sum256(data))
}

func (cw *CopyWorker) allGood(rpath string, resource *resources.ResourceInfo) bool {
	data, err := cw.readFile(rpath)
	if err != nil {
		return false
	}
	return cw.equal(cw.sha256sum(data), resource.SHA256)
}

func (cw *CopyWorker) ensureFor(name string, resource *resources.ResourceInfo) error {
	rpath := filepath.Join(cw.DestDir, name)
	if cw.allGood(rpath, resource) {
		return nil
	}
	filep, err := cw.open(name + ".gz")
	if err != nil {
		return err
	}
	defer filep.Close()
	gzfilep, err := cw.newReader(filep)
	if err != nil {
		return err
	}
	defer gzfilep.Close()
	data, err := cw.readAll(gzfilep)
	if err != nil {
		return err
	}
	if cw.different(cw.sha256sum(data), resource.SHA256) {
		return ErrSHA256Mismatch
	}
	return cw.writeFile(rpath, data, 0600)
}
