package nettests

import (
	"io/ioutil"
	"path"
	"testing"

	ooni "github.com/ooni/probe-cli"
	"github.com/ooni/probe-cli/internal/database"
	"github.com/ooni/probe-cli/internal/utils/shutil"
)

func newTestingContext(t *testing.T) *ooni.Context {
	homePath, err := ioutil.TempDir("", "ooniprobetests")
	if err != nil {
		t.Fatal(err)
	}
	configPath := path.Join(homePath, "config.json")
	testingConfig := path.Join("..", "..", "testdata", "testing-config.json")
	shutil.Copy(testingConfig, configPath, false)
	ctx := ooni.NewContext(configPath, homePath)
	swName := "ooniprobe-cli-tests"
	swVersion := "3.0.0-alpha"
	err = ctx.Init(swName, swVersion)
	if err != nil {
		t.Fatal(err)
	}
	return ctx
}

func TestCreateContext(t *testing.T) {
	newTestingContext(t)
}

func TestRun(t *testing.T) {
	ctx := newTestingContext(t)
	sess, err := ctx.NewSession()
	if err != nil {
		t.Fatal(err)
	}
	network, err := database.CreateNetwork(ctx.DB, sess)
	if err != nil {
		t.Fatal(err)
	}
	res, err := database.CreateResult(ctx.DB, ctx.Home, "middlebox", network.ID)
	if err != nil {
		t.Fatal(err)
	}
	nt := HTTPInvalidRequestLine{}
	ctl := NewController(nt, ctx, res, sess)
	nt.Run(ctl)
}
