// Package assetsx allows to manage assets.
package assetsx

import (
	"bytes"
	"compress/gzip"
	"io/ioutil"

	"github.com/ooni/probe-assets/assets"
	"github.com/ooni/probe-cli/v3/internal/engine/runtimex"
)

// manager is the assets manager.
type manager struct{}

// Must calls panic if we cannot read an asset.
func Must(data []byte, err error) []byte {
	runtimex.PanicOnError(err, "cannot read assets")
	return data
}

// ASNDatabaseData returns the ASN database data or an error.
func ASNDatabaseData() ([]byte, error) {
	return (&manager{}).read(assets.ASNDatabaseDataGzip())
}

// CountryDatabaseData returns the country database data or an error.
func CountryDatabaseData() ([]byte, error) {
	return (&manager{}).read(assets.CountryDatabaseDataGzip())
}

// read opens and reads the specified asset
func (m *manager) read(gzdata []byte) ([]byte, error) {
	gzfilep, err := gzip.NewReader(bytes.NewReader(gzdata))
	if err != nil {
		return nil, err
	}
	defer gzfilep.Close()
	return ioutil.ReadAll(gzfilep)
}
