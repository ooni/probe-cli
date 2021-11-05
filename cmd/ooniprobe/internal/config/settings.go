package config

// Sharing settings
type Sharing struct {
	UploadResults bool `json:"upload_results"`
}

// Advanced settings
type Advanced struct{}

// Nettests related settings
type Nettests struct {
	WebsitesMaxRuntime           int64    `json:"websites_max_runtime"`
	WebsitesURLLimit             int64    `json:"websites_url_limit"`
	WebsitesEnabledCategoryCodes []string `json:"websites_enabled_category_codes"`
}
