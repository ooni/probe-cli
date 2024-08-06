package engine_test

import (
	"context"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/engine"
	"github.com/ooni/probe-cli/v3/internal/kvstore"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/targetloading"
)

// This historical integration test ensures that we're able to fetch URLs from
// the dev infrastructure. We say this test's historical because the targetloading.Loader
// belonged to the engine package before we introduced richer input. It kind of feels
// good to keep this integration test here since we want to use a real session and a real
// Loader and double check whether we can get inputs. In a more distant future it would
// kind of make sense to have a broader package with this kind of integration tests.
func TestTargetLoaderInputOrQueryBackendWithNoInput(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	sess, err := engine.NewSession(context.Background(), engine.SessionConfig{
		AvailableProbeServices: []model.OOAPIService{{
			Address: "https://backend-hel.ooni.org/",
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
	il := &targetloading.Loader{
		InputPolicy: model.InputOrQueryBackend,
		Session:     sess,
	}
	ctx := context.Background()
	out, err := il.Load(ctx)
	// TODO(decfox): it seems `backend-hel.ooni.org` returns a different response
	// than intended which is why the test fails.
	if err != nil {
		t.Fatal(err)
	}
	if len(out) < 10 {
		// check-in SHOULD return AT LEAST 20 URLs at a time.
		t.Fatal("not the output length we expected")
	}
}
