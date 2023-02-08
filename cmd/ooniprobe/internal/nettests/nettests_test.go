package nettests

import (
	"context"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/ooni/probe-cli/v3/cmd/ooniprobe/internal/ooni"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func copyfile(source, dest string) error {
	data, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	return os.WriteFile(dest, data, 0600)
}

func newOONIProbe(t *testing.T) *ooni.Probe {
	homePath, err := ioutil.TempDir("", "ooniprobetests")
	if err != nil {
		t.Fatal(err)
	}
	configPath := path.Join(homePath, "config.json")
	testingConfig := path.Join("..", "..", "testdata", "testing-config.json")
	if err := copyfile(testingConfig, configPath); err != nil {
		t.Fatal(err)
	}
	probe := ooni.NewProbe(configPath, homePath)
	swName := "ooniprobe-cli-tests"
	swVersion := "3.0.0-alpha"
	err = probe.Init(swName, swVersion, "")
	if err != nil {
		t.Fatal(err)
	}
	return probe
}

func TestCreateContext(t *testing.T) {
	newOONIProbe(t)
}

func TestRun(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	probe := newOONIProbe(t)
	sess, err := probe.NewProbeEngine(context.Background(), model.RunTypeManual)
	if err != nil {
		t.Fatal(err)
	}
	db := probe.DB()
	network, err := db.CreateNetwork(sess)
	if err != nil {
		t.Fatal(err)
	}
	res, err := db.CreateResult(probe.Home(), "middlebox", network.ID)
	if err != nil {
		t.Fatal(err)
	}
	nt := HTTPInvalidRequestLine{}
	ctl := NewController(nt, probe, res, sess)
	nt.Run(ctl)
}
