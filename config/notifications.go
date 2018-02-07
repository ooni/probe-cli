package config

// Notifications settings
type Notifications struct {
	Enabled                bool `json:"enabled"`
	NotifyOnTestCompletion bool `json:"notify_on_test_completion"`
	NotifyOnNews           bool `json:"notify_on_news"`
}
