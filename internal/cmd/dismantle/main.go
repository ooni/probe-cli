package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/backendclient"
	"github.com/ooni/probe-cli/v3/internal/bytecounter"
	"github.com/ooni/probe-cli/v3/internal/experiment/webconnectivitylte"
	"github.com/ooni/probe-cli/v3/internal/geolocate"
	"github.com/ooni/probe-cli/v3/internal/kvstore"
	"github.com/ooni/probe-cli/v3/internal/logx"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/platform"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/sessionhttpclient"
	"github.com/ooni/probe-cli/v3/internal/sessionresolver"
	"github.com/ooni/probe-cli/v3/internal/tunnel"
	"github.com/ooni/probe-cli/v3/internal/version"
)

func main() {
	const softwareName = "dismantle"
	const softwareVersion = "0.1.0-dev"
	userAgent := fmt.Sprintf(
		"%s/%s ooniprobe-engine/%s",
		softwareName, softwareVersion,
		version.Version,
	)

	logHandler := logx.NewHandlerWithDefaultSettings()
	logHandler.Emoji = true
	logger := &log.Logger{Level: log.InfoLevel, Handler: logHandler}
	progressBar := model.NewPrinterCallbacks(logger)
	counter := bytecounter.New()
	home := filepath.Join(os.Getenv("HOME"), ".miniooni")
	statedir := filepath.Join(home, "kvstore2")
	ctx := context.Background()
	tunnelDir := filepath.Join(home, "tunnel")
	runtimex.Try0(os.MkdirAll(tunnelDir, 0700))

	kvstore := runtimex.Try1(kvstore.NewFS(statedir))

	tunnelConfig := &tunnel.Config{
		Name:      "tor",
		TunnelDir: tunnelDir,
		Logger:    logger,
	}
	tunnel, _ := runtimex.Try2(tunnel.Start(ctx, tunnelConfig))
	defer tunnel.Stop()
	proxyURL := tunnel.SOCKS5ProxyURL()

	sessionResolver := &sessionresolver.Resolver{
		ByteCounter: counter,
		KVStore:     kvstore,
		Logger:      logger,
		ProxyURL:    proxyURL,
	}
	defer sessionResolver.CloseIdleConnections()

	geolocateConfig := geolocate.Config{
		Resolver:  sessionResolver,
		Logger:    logger,
		UserAgent: model.HTTPHeaderUserAgent,
	}
	geolocateTask := geolocate.NewTask(geolocateConfig) // XXX
	location := runtimex.Try1(geolocateTask.Run(ctx))
	logger.Infof("%+v", location)

	sessionHTTPClientConfig := &sessionhttpclient.Config{
		ByteCounter: counter,
		Logger:      logger,
		Resolver:    sessionResolver,
		ProxyURL:    proxyURL,
	}
	sessionHTTPClient := sessionhttpclient.New(sessionHTTPClientConfig)
	defer sessionHTTPClient.CloseIdleConnections()

	backendClientConfig := &backendclient.Config{
		KVStore:    kvstore,
		HTTPClient: sessionHTTPClient,
		Logger:     logger,
		UserAgent:  userAgent,
		BaseURL:    nil,
	}
	backendClient := backendclient.New(backendClientConfig)

	checkInConfig := &model.OOAPICheckInConfig{
		Charging:        false,
		OnWiFi:          false,
		Platform:        platform.Name(),
		ProbeASN:        location.ProbeASNString(),
		ProbeCC:         location.CountryCode,
		RunType:         "manual",
		SoftwareName:    softwareName,
		SoftwareVersion: softwareName,
		WebConnectivity: model.OOAPICheckInConfigWebConnectivity{
			CategoryCodes: []string{},
		},
	}
	checkInResult := runtimex.Try1(backendClient.CheckIn(ctx, checkInConfig))
	logger.Infof("%+v", checkInResult)

	runtimex.Assert(checkInResult.Tests.WebConnectivity != nil, "no web connectivity info")
	reportID := checkInResult.Tests.WebConnectivity.ReportID

	experimentSession := &experimentSession{
		httpClient:  sessionHTTPClient,
		location:    location,
		logger:      logger,
		testHelpers: checkInResult.Conf.TestHelpers,
		userAgent:   userAgent,
	}

	testStartTime := time.Now()
	for _, input := range checkInResult.Tests.WebConnectivity.URLs {
		cfg := &webconnectivitylte.Config{}
		runner := webconnectivitylte.NewExperimentMeasurer(cfg)
		measurement := model.NewMeasurement(
			location, runner.ExperimentName(), runner.ExperimentVersion(),
			testStartTime, reportID, softwareName, softwareVersion, input.URL,
		)
		args := &model.ExperimentArgs{
			Callbacks:   progressBar,
			Measurement: measurement,
			Session:     experimentSession,
		}
		if err := runner.Run(ctx, args); err != nil {
			logger.Warnf("runner.Run failed: %s", err.Error())
		}
		if err := backendClient.Submit(ctx, measurement); err != nil {
			logger.Warnf("backendClient.Submit failed: %s", err.Error())
		}
		log.Infof("measurement URL: %s", makeExplorerURL(reportID, input.URL))
	}
}

func makeExplorerURL(reportID, input string) string {
	query := url.Values{}
	query.Add("input", input)
	explorerURL := &url.URL{
		Scheme:      "https",
		Host:        "explorer.ooni.org",
		Path:        fmt.Sprintf("/measurement/%s", reportID),
		RawQuery:    query.Encode(),
		Fragment:    "",
		RawFragment: "",
	}
	return explorerURL.String()
}
