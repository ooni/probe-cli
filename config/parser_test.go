package config

import (
	"testing"
)

func TestParseConfig(t *testing.T) {
	config, err := ReadConfig("testdata/valid-config.json")
	if err != nil {
		t.Error(err)
	}

	if len(config.NettestGroups.Middlebox.EnabledTests) < 0 {
		t.Error("at least one middlebox test should be enabled")
	}
	if config.Advanced.IncludeCountry == false {
		t.Error("country should be included")
	}
}
