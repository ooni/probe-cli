package main

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/geolocate"
)

func init() {
	taskRegistry["Geolocate"] = &geolocateTaskRunner{}
}

// geolocateOptions contains the request arguments for the Geolocate task.
type geolocateOptions struct {
	SessionId int64 `json:",omitempty"`
}

// geolocateResponse is the response for the Geolocate task.
type geolocateResponse struct {
	ASN                 uint   `json:",omitempty"`
	CountryCode         string `json:",omitempty"`
	NetworkName         string `json:",omitempty"`
	ProbeIP             string `json:",omitempty"`
	ResolverASN         uint   `json:",omitempty"`
	ResolverIP          string `json:",omitempty"`
	ResolverNetworkName string `json:",omitempty"`
	Error               string `json:",omitempty"`
}

type geolocateTaskRunner struct{}

var _ taskRunner = &geolocateTaskRunner{}

// main implements taskRunner.main
func (tr *geolocateTaskRunner) main(ctx context.Context,
	emitter taskMaybeEmitter, req *request, resp *response) {
	logger := newTaskLogger(emitter)
	sessionId := req.Geolocate.SessionId
	sess := mapSession[sessionId]
	if sess == nil {
		logger.Warnf("session: %s", errInvalidSessionId.Error())
		resp.Geolocate.Error = errInvalidSessionId.Error()
		return
	}
	gt := geolocate.NewTask(geolocate.Config{
		Logger:    sess.Logger(),
		Resolver:  sess.Resolver(),
		UserAgent: sess.UserAgent(),
	})
	results, err := gt.Run(ctx)
	if err != nil {
		logger.Warnf("geolocate: %s", err.Error())
		resp.Geolocate.Error = err.Error()
		return
	}
	resp = &response{
		Geolocate: geolocateResponse{
			ASN:                 results.ASN,
			CountryCode:         results.CountryCode,
			NetworkName:         results.NetworkName,
			ProbeIP:             results.ProbeIP,
			ResolverASN:         results.ResolverASN,
			ResolverIP:          results.ResolverIP,
			ResolverNetworkName: results.ResolverNetworkName,
		},
	}
}
