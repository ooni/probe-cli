package echcheck

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"net"
	"net/url"
	"time"

	"github.com/ooni/probe-cli/v3/internal/logx"
	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func connectAndHandshake(ctx context.Context, echConfigList []byte, isGrease bool, startTime time.Time, address string, target *url.URL, outerServerName string, logger model.Logger) (chan model.ArchivalTLSOrQUICHandshakeResult, error) {

	channel := make(chan model.ArchivalTLSOrQUICHandshakeResult)

	ol := logx.NewOperationLogger(logger, "echcheck: TCPConnect %s", address)
	var dialer net.Dialer
	conn, err := dialer.DialContext(ctx, "tcp", address)
	ol.Stop(err)
	if err != nil {
		return nil, netxlite.NewErrWrapper(netxlite.ClassifyGenericError, netxlite.ConnectOperation, err)
	}

	tlsConfig := genEchTLSConfig(target.Hostname(), echConfigList)

	go func() {
		hs := handshake(ctx, conn, echConfigList, isGrease, startTime, address, logger, tlsConfig)
		hs.OuterServerName = outerServerName
		channel <- *hs
	}()

	return channel, nil
}

func handshake(ctx context.Context, conn net.Conn, echConfigList []byte, isGrease bool, startTime time.Time, address string, logger model.Logger, tlsConfig *tls.Config) *model.ArchivalTLSOrQUICHandshakeResult {
	var d string
	if isGrease {
		d = " (GREASE)"
	} else if len(echConfigList) > 0 {
		d = " (RealECH)"
	}
	ol := logx.NewOperationLogger(logger, "echcheck: DialTLS%s", d)
	start := time.Now()

	maybeTLSConn := tls.Client(conn, tlsConfig)
	err := maybeTLSConn.HandshakeContext(ctx)

	if echErr, ok := err.(*tls.ECHRejectionError); ok && isGrease {
		if len(echErr.RetryConfigList) > 0 {
			tlsConfig.EncryptedClientHelloConfigList = echErr.RetryConfigList
			maybeTLSConn, err = tls.Dial("tcp", address, tlsConfig)
		}
	}
	finish := time.Now()
	ol.Stop(err)

	var connState tls.ConnectionState
	// If there's been an error, processing maybeTLSConn can panic.
	if err != nil {
		connState = tls.ConnectionState{}
	} else {
		connState = netxlite.MaybeTLSConnectionState(maybeTLSConn)
	}
	hs := measurexlite.NewArchivalTLSOrQUICHandshakeResult(0, start.Sub(startTime),
		"tcp", address, tlsConfig, connState, err, finish.Sub(startTime))
	if isGrease {
		hs.ECHConfig = "GREASE"
	} else {
		hs.ECHConfig = base64.StdEncoding.EncodeToString(echConfigList)
	}
	return hs
}

func genEchTLSConfig(host string, echConfigList []byte) *tls.Config {
	if len(echConfigList) == 0 {
		return &tls.Config{ServerName: host}
	}
	return &tls.Config{
		EncryptedClientHelloConfigList: echConfigList,
		// This will be used as the inner SNI and we will validate
		// we get a certificate for this name.  The outer SNI will
		// be set based on the ECH config.
		ServerName: host,
	}
}
