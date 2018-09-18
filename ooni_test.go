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
	if err := ctx.Init(); err != nil {
		t.Error(err)
		t.Fatal("failed to init the context")
	}

	configPath := path.Join(ooniHome, "config.json")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("config file was not created")
	}
}
