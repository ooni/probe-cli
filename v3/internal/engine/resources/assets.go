package resources

const (
	// Version contains the assets version.
	Version = 20210129095811

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
		URLPath:  "/ooni/probe-assets/releases/download/20210129095811/asn.mmdb.gz",
		GzSHA256: "ef1759bf8b77128723436c4ec5a3d7f2e695fb5a959e741ba39012ced325132c",
		SHA256:   "0afa5afc48ba913933f17b11213c3044499c8338cf63b8f9af2778faa5875474",
	},
	"country.mmdb": {
		URLPath:  "/ooni/probe-assets/releases/download/20210129095811/country.mmdb.gz",
		GzSHA256: "5d465224ab02242a8a79652161d2768e64dd91fc1ed840ca3d0746f4cd29a914",
		SHA256:   "b4aa1292d072d9b2631711e6d3ac69c1e89687b4d513d43a1c330a92b7345e4d",
	},
}
