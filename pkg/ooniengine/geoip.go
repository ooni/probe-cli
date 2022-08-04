package main

//
// GeoIP task
//

import (
	"context"
	"encoding/json"
)

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
	var config GeoIPConfig
	if err := json.Unmarshal(args, &config); err != nil {
		logger.Warnf("geoip: cannot parse settings: %s", err.Error())
		return
	}
	logger.verbose = config.Session.LogLevel == LogLevelDebug
	sess, err := newSession(ctx, &config.Session, logger)
	if err != nil {
		logger.Warnf("geoip: cannot create a new session: %s", err.Error())
		return
	}
	defer sess.Close()
	if err := sess.MaybeLookupLocationContext(ctx); err != nil {
		logger.Warnf("geoip: cannot lookup location: %s", err.Error())
		return
	}
	event := &GeoIPEventValue{
		Failure:             newFailureString(err),
		ProbeIP:             sess.ProbeIP(),
		ProbeASN:            sess.ProbeASNString(),
		ProbeCC:             sess.ProbeCC(),
		ProbeNetworkName:    sess.ProbeNetworkName(),
		ResolverIP:          sess.ResolverIP(),
		ResolverASN:         sess.ResolverASNString(),
		ResolverNetworkName: sess.ResolverNetworkName(),
	}
	emitter.maybeEmitEvent(GeoIPEventName, event)
}
