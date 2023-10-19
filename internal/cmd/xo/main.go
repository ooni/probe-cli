package main

import (
	"context"
	"os"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/engine"
	"github.com/ooni/probe-cli/v3/internal/enginerun"
	"github.com/ooni/probe-cli/v3/internal/hujsonx"
	"github.com/ooni/probe-cli/v3/internal/kvstore"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/version"
)

func main() {
	rawNettest := runtimex.Try1(os.ReadFile(os.Args[1]))
	var nt enginerun.Nettest
	runtimex.Try0(hujsonx.Unmarshal(rawNettest, &nt))

	ctx := context.Background()

	//log.SetLevel(log.DebugLevel)

	config := engine.SessionConfig{
		AvailableProbeServices: []model.OOAPIService{},
		KVStore:                &kvstore.Memory{},
		Logger:                 log.Log,
		ProxyURL:               nil,
		SoftwareName:           "miniooni",
		SoftwareVersion:        version.Version,
		TempDir:                "/tmp",
		TorArgs:                []string{},
		TorBinary:              "",
		SnowflakeRendezvous:    "",
		TunnelDir:              "xo_tunnel_dir",
	}
	sess := runtimex.Try1(engine.NewSession(ctx, config))

	// Note: we need to lookup backends and test helpers in this case
	// because otherwise we cannot run web_connectivity
	//
	// XXX: ideally this would also call the check-in API
	runtimex.Try0(sess.MaybeLookupBackends())
	runtimex.Try0(sess.MaybeLookupLocation())

	// while this API may be a bit weird, we have basically reimplemented miniooni in 50 LoC
	submitter := runtimex.Try1(engine.NewSubmitter(ctx, engine.SubmitterConfig{
		Enabled: true,
		Session: sess,
		Logger:  log.Log,
	}))

	// run the nettest in a background goroutine and handle the generated events
	events := runtimex.Try1(enginerun.Start(ctx, sess, &nt))
	for {
		select {
		case <-events.Done():
			return

		case dataUsage := <-events.DataUsage():
			log.Infof("data usage: %+v", dataUsage)

		case runError := <-events.RunError():
			log.Warnf("experiment failed: %s", runError.Err.Error())

		case runSuccess := <-events.RunSuccess():
			runtimex.Try0(submitter.Submit(ctx, runSuccess.Measurement))
		}
	}
}
