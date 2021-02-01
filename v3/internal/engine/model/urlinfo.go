package model

// URLInfo contains info on a test lists URL
type URLInfo struct {
	CategoryCode string `json:"category_code"`
	CountryCode  string `json:"country_code"`
	URL          string `json:"url"`
}
