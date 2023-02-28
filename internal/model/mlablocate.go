package model

//
// M-Lab locate v2 API model
//

// TODO(bassosimone): before merging, make sure internal/mlablocatev2
// uses these definitions rather than its own private ones.

// MLabLocateServerLocation is the location of an m-lab server.
type MLabLocateServerLocation struct {
	// City is the city where the server is deployed.
	City string `json:"city"`

	// Country is the country where the server is deployed.
	Country string `json:"country"`
}

// MLabLocateSingleResult is a single result in [MLabLocateResults].
type MLabLocateSingleResult struct {
	// Machine is the name of the machine.
	Machine string `json:"machine"`

	// Location contains the location of the machine.
	Location MLabLocateServerLocation `json:"location"`

	// URLs contains the URLs to use.
	URLs map[string]string `json:"urls"`
}

// MLabLocateResults is the JSON returned by m-lab locate v2 API.
type MLabLocateResults struct {
	// Results contains the results to return.
	Results []MLabLocateSingleResult `json:"results"`
}
