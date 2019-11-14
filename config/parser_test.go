package config

import (
	"testing"
)

func TestParseConfig(t *testing.T) {
	config, err := ReadConfig("testdata/valid-config.json")
	if err != nil {
		t.Error(err)
	}

	if config.Sharing.IncludeCountry == false {
		t.Error("country should be included")
	}
}
