package model

// OOAPIURLInfo contains info on a test lists URL
type OOAPIURLInfo struct {
	CategoryCode string `json:"category_code"`
	CountryCode  string `json:"country_code"`
	URL          string `json:"url"`
}
