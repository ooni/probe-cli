package engine_test

import (
	"context"
	"testing"

	"github.com/apex/log"
	engine "github.com/ooni/probe-cli/v3/internal/engine"
	"github.com/ooni/probe-cli/v3/internal/engine/model"
	"github.com/ooni/probe-cli/v3/internal/kvstore"
)

func TestInputLoaderInputOrQueryBackendWithNoInput(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	sess, err := engine.NewSession(context.Background(), engine.SessionConfig{
		AvailableProbeServices: []model.OOAPIService{{
			Address: "https://ams-pg-test.ooni.org/",
			Type:    "https",
		}},
		KVStore:         &kvstore.Memory{},
		Logger:          log.Log,
		SoftwareName:    "miniooni",
		SoftwareVersion: "0.1.0-dev",
		TempDir:         "testdata",
	})
	if err != nil {
		t.Fatal(err)
	}
	defer sess.Close()
	il := &engine.InputLoader{
		InputPolicy: engine.InputOrQueryBackend,
		Session:     sess,
	}
	ctx := context.Background()
	out, err := il.Load(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) < 10 {
		// check-in SHOULD return AT LEAST 20 URLs at a time.
		t.Fatal("not the output length we expected")
	}
}
