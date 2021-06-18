package config

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io/ioutil"
	"os"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/iox"
)

func getShasum(path string) (string, error) {
	hasher := sha256.New()

	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := iox.CopyContext(context.Background(), hasher, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func TestParseConfig(t *testing.T) {
	config, err := ReadConfig("testdata/valid-config.json")
	if err != nil {
		t.Error(err)
	}
	if config.Sharing.UploadResults != true {
		t.Fatal("not the expected value for UploadResults")
	}
}

func TestUpdateConfig(t *testing.T) {
	tmpFile, err := ioutil.TempFile(os.TempDir(), "ooniconfig-")
	if err != nil {
		t.Error(err)
	}
	configPath := tmpFile.Name()
	defer os.Remove(configPath)

	data, err := os.ReadFile("testdata/config-v0.json")
	if err != nil {
		t.Error(err)
	}
	err = os.WriteFile(configPath, data, 0644)
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
