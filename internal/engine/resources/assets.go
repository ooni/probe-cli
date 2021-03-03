package resources

const (
	// Version contains the assets version.
	Version = 20210303114512

	// ASNDatabaseName is the ASN-DB file name
	ASNDatabaseName = "asn.mmdb"

	// CountryDatabaseName is country-DB file name
	CountryDatabaseName = "country.mmdb"

	// BaseURL is the asset's repository base URL
	BaseURL = "https://github.com/"
)

// ResourceInfo contains information on a resource.
type ResourceInfo struct {
	// URLPath is the resource's URL path.
	URLPath string

	// GzSHA256 is used to validate the downloaded file.
	GzSHA256 string

	// SHA256 is used to check whether the assets file
	// stored locally is still up-to-date.
	SHA256 string
}

// All contains info on all known assets.
var All = map[string]ResourceInfo{
	"asn.mmdb": {
		URLPath:  "/ooni/probe-assets/releases/download/20210303114512/asn.mmdb.gz",
		GzSHA256: "efafd5a165c5a4e6bf6258d87ed685254a2660669eb4557e25c5ed72e48d039a",
		SHA256:   "675dbaec3fa1e6f12957c4e4ddee03f50f5192507b5095ccb9ed057468c2441b",
	},
	"country.mmdb": {
		URLPath:  "/ooni/probe-assets/releases/download/20210303114512/country.mmdb.gz",
		GzSHA256: "7f1db0e2903271258319834f26bbcdedd2d0641457a8c0a63b048a985b7d6e7b",
		SHA256:   "19e4d2c5cd31789da1a67baf883995f2ea03c4b8ba7342b69ef8ae2c2aa8409c",
	},
}
