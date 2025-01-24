package echcheck

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"net"
	"net/url"
	"time"

	"github.com/ooni/probe-cli/v3/internal/logx"
	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func connectAndHandshake(ctx context.Context, echConfigList []byte, isGrease bool, startTime time.Time, address string, target *url.URL, outerServerName string, logger model.Logger, testOnlyRootCAs *x509.CertPool) (chan TestKeys, error) {

	channel := make(chan TestKeys)

	ol := logx.NewOperationLogger(logger, "echcheck: TCPConnect %s", address)
	trace := measurexlite.NewTrace(0, startTime)
	dialer := trace.NewDialerWithoutResolver(logger)
	conn, err := dialer.DialContext(ctx, "tcp", address)
	ol.Stop(err)
	if err != nil {
		return nil, netxlite.NewErrWrapper(netxlite.ClassifyGenericError, netxlite.ConnectOperation, err)
	}

	tlsConfig := genEchTLSConfig(target.Hostname(), echConfigList, testOnlyRootCAs)

	go func() {
		tk := handshake(ctx, conn, echConfigList, isGrease, startTime, address, logger, tlsConfig)
		tk.TLSHandshakes[0].OuterServerName = outerServerName
		tcpcs := trace.TCPConnects()
		tk.TCPConnect = append(tk.TCPConnect, tcpcs...)
		channel <- tk
	}()

	return channel, nil
}

func handshake(ctx context.Context, conn net.Conn, echConfigList []byte, isGrease bool, startTime time.Time, address string, logger model.Logger, tlsConfig *tls.Config) TestKeys {
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
			// TODO: trace this TCP connection
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
	tk := TestKeys{
		TLSHandshakes: []*model.ArchivalTLSOrQUICHandshakeResult{hs},
	}
	return tk
}

func genEchTLSConfig(host string, echConfigList []byte, testOnlyRootCAs *x509.CertPool) *tls.Config {
	c := &tls.Config{ServerName: host}
	if len(echConfigList) > 0 {
		c.EncryptedClientHelloConfigList = echConfigList
	}
	if testOnlyRootCAs != nil {
		c.RootCAs = testOnlyRootCAs
	}
	return c
}
