// Package assetsdir contains code to cleanup the assets dir. We removed
// the assetsdir in the 3.9.0 development cycle.
package assetsdir

import (
	"errors"
	"os"
	"path/filepath"
)

// ErrEmptyDir indicates that you passed to Cleanup an empty dir.
var ErrEmptyDir = errors.New("empty assets directory")

// Result is the result of a Cleanup run.
type Result struct {
	// ASNDatabaseErr is the error of deleting the
	// file containing the old ASN database.
	ASNDatabaseErr error

	// CABundleErr is the error of deleting the file
	// containing the old CA bundle.
	CABundleErr error

	// CountryDatabaseErr is the error of deleting the
	// file containing the old country database.
	CountryDatabaseErr error

	// RmdirErr is the error of deleting the supposedly
	// empty directory that contained assets.
	RmdirErr error
}

// Cleanup removes data from the assetsdir. This function will
// try to delete the known assets inside of dir. It then also
// tries to delete the directory. If the directory is not empty,
// this operation will fail. That means the user has put some
// extra data in there and we don't want to remove it.
//
// Returns the Result of cleaning up the assets on success and
// an error on failure. The only cause of error is passing to
// this function an empty directory. The Result data structure
// contains the result of each individual remove operation.
func Cleanup(dir string) (*Result, error) {
	return fcleanup(dir, os.Remove)
}

// fcleanup is a version of Cleanup where we can mock the real function
// used for removing files and dirs, so we can write unit tests.
func fcleanup(dir string, remove func(name string) error) (*Result, error) {
	if dir == "" {
		return nil, ErrEmptyDir
	}
	r := &Result{}
	asndb := filepath.Join(dir, "asn.mmdb")
	r.ASNDatabaseErr = os.Remove(asndb)
	cabundle := filepath.Join(dir, "ca-bundle.pem")
	r.CABundleErr = os.Remove(cabundle)
	countrydb := filepath.Join(dir, "country.mmdb")
	r.CountryDatabaseErr = os.Remove(countrydb)
	r.RmdirErr = os.Remove(dir)
	return r, nil
}
