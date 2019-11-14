package config

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"io/ioutil"
	"os"
	"testing"
)

func getShasum(path string) (string, error) {
	hasher := sha256.New()

	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := io.Copy(hasher, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func TestParseConfig(t *testing.T) {
	config, err := ReadConfig("testdata/valid-config.json")
	if err != nil {
		t.Error(err)
	}

	if config.Sharing.IncludeCountry == false {
		t.Error("country should be included")
	}
}

func TestUpdateConfig(t *testing.T) {
	tmpFile, err := ioutil.TempFile(os.TempDir(), "ooniconfig-")
	if err != nil {
		t.Error(err)
	}
	configPath := tmpFile.Name()
	defer os.Remove(configPath)

	data, err := ioutil.ReadFile("testdata/config-v0.json")
	if err != nil {
		t.Error(err)
	}
	err = ioutil.WriteFile(configPath, data, 0644)
	if err != nil {
		t.Error(err)
	}

	origShasum, err := getShasum(configPath)
	if err != nil {
		t.Error(err)
	}
	config, err := ReadConfig(configPath)
	if err != nil {
		t.Error(err)
	}
	origIncludeIP := config.Sharing.IncludeIP
	origIncludeASN := config.Sharing.IncludeASN
	origIncludeCountry := config.Sharing.IncludeCountry
	origUploadResults := config.Sharing.UploadResults
	origInformedConsent := config.InformedConsent
	if err != nil {
		t.Error(err)
	}

	config.MaybeMigrate()
	migratedShasum, err := getShasum(configPath)
	if err != nil {
		t.Error(err)
	}
	if migratedShasum == origShasum {
		t.Fatal("the config was not migrated")
	}

	newConfig, err := ReadConfig(configPath)
	if err != nil {
		t.Error(err)
	}
	if newConfig.Sharing.IncludeIP != origIncludeIP {
		t.Error("includeIP differs")
	}
	if newConfig.Sharing.IncludeASN != origIncludeASN {
		t.Error("includeASN differs")
	}
	if newConfig.Sharing.IncludeCountry != origIncludeCountry {
		t.Error("includeCountry differs")
	}
	if newConfig.Sharing.UploadResults != origUploadResults {
		t.Error("UploadResults differs")
	}
	if newConfig.InformedConsent != origInformedConsent {
		t.Error("InformedConsent differs")
	}

	// Check that the config file stays the same if it's already the most up to
	// date version
	config.MaybeMigrate()
	finalShasum, err := getShasum(configPath)
	if err != nil {
		t.Error(err)
	}
	if migratedShasum != finalShasum {
		t.Fatal("the config was migrated again")
	}
}
