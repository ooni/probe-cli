package model

// URLListConfig contains configuration for fetching the URL list.
type URLListConfig struct {
	Categories  []string // Categories to query for (empty means all)
	CountryCode string   // CountryCode is the optional country code
	Limit       int64    // Max number of URLs (<= 0 means no limit)
}
