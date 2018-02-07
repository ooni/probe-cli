package config

// AutomatedTesting settings
type AutomatedTesting struct {
	Enabled          bool     `json:"enabled"`
	EnabledTests     []string `json:"enabled_tests"`
	MonthlyAllowance string   `json:"monthly_allowance"`
}
