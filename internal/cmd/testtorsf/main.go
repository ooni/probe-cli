//go:build ooni_libtor

package main

import (
	"context"
	"net/http"
	"time"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/experiment/vanillator"
	"github.com/ooni/probe-cli/v3/internal/kvstore"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

func runit() {
	measurer := vanillator.NewExperimentMeasurer(vanillator.Config{
		//DisablePersistentDatadir: false,
		DisableProgress: false,
		//RendezvousMethod:         "",
	})
	meas := &model.Measurement{}
	err := measurer.Run(
		context.Background(),
		&model.ExperimentArgs{
			Callbacks:   model.NewPrinterCallbacks(model.DiscardLogger),
			Measurement: meas,
			Session: &mocks.Session{
				MockDefaultHTTPClient: func() model.HTTPClient {
					return http.DefaultClient
				},
				MockKeyValueStore: func() model.KeyValueStore {
					return &kvstore.Memory{}
				},
				MockLogger: func() model.Logger {
					return log.Log
				},
				MockSoftwareName: func() string {
					return "miniooni"
				},
				MockSoftwareVersion: func() string {
					return "0.1.0-dev"
				},
				MockTempDir: func() string {
					return "x/tmp"
				},
				MockTunnelDir: func() string {
					return "x/tunnel"
				},
				MockUserAgent: func() string {
					return model.HTTPHeaderUserAgent
				},
			},
		},
	)
	runtimex.PanicOnError(err, "measurer.Run failed")
	tk := meas.TestKeys.(*vanillator.TestKeys)
	runtimex.Assert(tk.Success, "did not succeed")
}

func main() {
	for {
		runit()
		log.Info("************* now let's wait a bit ********************************")
		time.Sleep(45 * time.Second)
	}
}
