package ooni

import (
	"io/ioutil"
	"os"
	"path"
	"testing"
)

func TestInit(t *testing.T) {
	ooniHome, err := ioutil.TempDir("", "oonihome")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(ooniHome)

	ctx := NewContext("", ooniHome)
	swName := "ooniprobe-cli-tests"
	swVersion := "3.0.0-alpha"
	if err := ctx.Init(swName, swVersion); err != nil {
		t.Error(err)
		t.Fatal("failed to init the context")
	}

	configPath := path.Join(ooniHome, "config.json")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("config file was not created")
	}
}
