package config

type websiteSettings struct {
	EnabledCategories []string `json:"enabled_categories"`
}

type instantMessagingSettings struct {
	EnabledTests []string `json:"enabled_tests"`
}

type performanceSettings struct {
	EnabledTests   []string `json:"enabled_tests"`
	NDTServer      string   `json:"ndt_server"`
	NDTServerPort  string   `json:"ndt_server_port"`
	DashServer     string   `json:"dash_server"`
	DashServerPort string   `json:"dash_server_port"`
}

type middleboxSettings struct {
	EnabledTests []string `json:"enabled_tests"`
}

// NettestGroups related settings
type NettestGroups struct {
	Websites         websiteSettings          `json:"websites"`
	InstantMessaging instantMessagingSettings `json:"instant_messaging"`
	Performance      performanceSettings      `json:"performance"`
	Middlebox        middleboxSettings        `json:"middlebox"`
}
