package ndt7

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/url"

	"github.com/gorilla/websocket"
	"github.com/ooni/probe-cli/v3/internal/engine/model"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/dialer"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/resolver"
)

type dialManager struct {
	ndt7URL         string
	logger          model.Logger
	proxyURL        *url.URL
	readBufferSize  int
	tlsConfig       *tls.Config
	userAgent       string
	writeBufferSize int
}

func newDialManager(ndt7URL string, logger model.Logger, userAgent string) dialManager {
	return dialManager{
		ndt7URL:         ndt7URL,
		logger:          logger,
		readBufferSize:  paramMaxBufferSize,
		userAgent:       userAgent,
		writeBufferSize: paramMaxBufferSize,
	}
}

func (mgr dialManager) dialWithTestName(ctx context.Context, testName string) (*websocket.Conn, error) {
	var reso resolver.Resolver = resolver.SystemResolver{}
	reso = resolver.LoggingResolver{Resolver: reso, Logger: mgr.logger}
	var dlr dialer.Dialer = dialer.Default
	dlr = dialer.TimeoutDialer{Dialer: dlr}
	dlr = dialer.ErrorWrapperDialer{Dialer: dlr}
	dlr = dialer.LoggingDialer{Dialer: dlr, Logger: mgr.logger}
	dlr = dialer.DNSDialer{Dialer: dlr, Resolver: reso}
	dlr = dialer.ProxyDialer{Dialer: dlr, ProxyURL: mgr.proxyURL}
	dlr = dialer.ByteCounterDialer{Dialer: dlr}
	dlr = dialer.ShapingDialer{Dialer: dlr}
	dialer := websocket.Dialer{
		NetDialContext:  dlr.DialContext,
		ReadBufferSize:  mgr.readBufferSize,
		TLSClientConfig: mgr.tlsConfig,
		WriteBufferSize: mgr.writeBufferSize,
	}
	headers := http.Header{}
	headers.Add("Sec-WebSocket-Protocol", "net.measurementlab.ndt.v7")
	headers.Add("User-Agent", mgr.userAgent)
	mgr.logrequest(mgr.ndt7URL, headers)
	conn, _, err := dialer.DialContext(ctx, mgr.ndt7URL, headers)
	mgr.logresponse(err)
	return conn, err
}

func (mgr dialManager) logrequest(url string, headers http.Header) {
	mgr.logger.Debugf("> GET %s", url)
	for key, values := range headers {
		for _, v := range values {
			mgr.logger.Debugf("> %s: %s", key, v)
		}
	}
	mgr.logger.Debug("> Connection: upgrade")
	mgr.logger.Debug("> Upgrade: websocket")
	mgr.logger.Debug(">")
}

func (mgr dialManager) logresponse(err error) {
	if err != nil {
		mgr.logger.Debugf("< %+v", err)
		return
	}
	mgr.logger.Debug("< 101")
	mgr.logger.Debug("< Connection: upgrade")
	mgr.logger.Debug("< Upgrade: websocket")
	mgr.logger.Debug("<")
}

func (mgr dialManager) dialDownload(ctx context.Context) (*websocket.Conn, error) {
	return mgr.dialWithTestName(ctx, "download")
}

func (mgr dialManager) dialUpload(ctx context.Context) (*websocket.Conn, error) {
	return mgr.dialWithTestName(ctx, "upload")
}
