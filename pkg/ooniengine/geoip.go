package main

//
// GeoIP task
//

import (
	"context"

	"github.com/ooni/probe-cli/v3/pkg/ooniengine/abi"
	"google.golang.org/protobuf/proto"
)

func init() {
	taskRegistry["GeoIP"] = newGeoIPRunner()
}

// newGeoIPRunner creates a new geoIPRunner.
func newGeoIPRunner() taskRunner {
	return &geoIPRunner{}
}

// geoIPRunner is the geoip task runner.
type geoIPRunner struct{}

var _ taskRunner = &geoIPRunner{}

// main implements taskRunner.main.
func (r *geoIPRunner) main(ctx context.Context, emitter taskMaybeEmitter, args []byte) {
	logger := newTaskLogger(emitter)
	var config abi.GeoIPConfig
	if err := proto.Unmarshal(args, &config); err != nil {
		logger.Warnf("geoip: cannot parse settings: %s", err.Error())
		return
	}
	// 🔥🔥🔥 Rule of thumb when reviewing protobuf code: if the code is using
	// the safe GetXXX accessors, it's good, otherwise it's not good
	logger.verbose = config.GetSession().GetLogLevel() == abi.LogLevel_DEBUG
	sess, err := newSession(ctx, config.GetSession(), logger)
	if err != nil {
		logger.Warnf("geoip: cannot create a new session: %s", err.Error())
		return
	}
	defer sess.Close()
	if err := sess.MaybeLookupLocationContext(ctx); err != nil {
		logger.Warnf("geoip: cannot lookup location: %s", err.Error())
		return
	}
	event := &abi.GeoIPEvent{
		Failure:             newFailureString(err),
		ProbeIp:             sess.ProbeIP(),
		ProbeAsn:            sess.ProbeASNString(),
		ProbeCc:             sess.ProbeCC(),
		ProbeNetworkName:    sess.ProbeNetworkName(),
		ResolverIp:          sess.ResolverIP(),
		ResolverAsn:         sess.ResolverASNString(),
		ResolverNetworkName: sess.ResolverNetworkName(),
	}
	emitter.maybeEmitEvent("GeoIP", event)
}
