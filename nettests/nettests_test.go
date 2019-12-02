package nettests

import (
	"io/ioutil"
	"path"
	"testing"

	ooni "github.com/ooni/probe-cli"
	"github.com/ooni/probe-cli/internal/database"
	"github.com/ooni/probe-cli/utils/shutil"
)

func newTestingContext(t *testing.T) *ooni.Context {
	homePath, err := ioutil.TempDir("", "ooniprobetests")
	if err != nil {
		t.Fatal(err)
	}
	configPath := path.Join(homePath, "config.json")
	testingConfig := path.Join("..", "testdata", "testing-config.json")
	shutil.Copy(testingConfig, configPath, false)
	ctx := ooni.NewContext(configPath, homePath)
	err = ctx.Init()
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
	network, err := database.CreateNetwork(ctx.DB, ctx.Session)
	if err != nil {
		t.Fatal(err)
	}
	res, err := database.CreateResult(ctx.DB, ctx.Home, "im", network.ID)
	if err != nil {
		t.Fatal(err)
	}
	nt := Telegram{}
	ctl := NewController(nt, ctx, res)
	nt.Run(ctl)
}
