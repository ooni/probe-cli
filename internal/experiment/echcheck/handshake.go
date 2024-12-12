package echcheck

import (
	"context"
	"crypto/tls"
	"net"
	"time"

	"github.com/ooni/probe-cli/v3/internal/logx"
	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

const echExtensionType uint16 = 0xfe0d

func connectAndHandshake(
	ctx context.Context,
	trace *measurexlite.Trace,
	startTime time.Time,
	address string, sni string, outerSni string,
	logger model.Logger) (chan model.ArchivalTLSOrQUICHandshakeResult, error) {

	channel := make(chan model.ArchivalTLSOrQUICHandshakeResult)

	ol := logx.NewOperationLogger(logger, "echcheck: TCPConnect %s", address)
	var dialer net.Dialer
	conn, err := dialer.DialContext(ctx, "tcp", address)
	ol.Stop(err)
	if err != nil {
		return nil, netxlite.NewErrWrapper(netxlite.ClassifyGenericError, netxlite.ConnectOperation, err)
	}

	go func() {
		var res *model.ArchivalTLSOrQUICHandshakeResult
		if outerSni == "" {
			res = handshake(
				ctx,
				trace,
				conn,
				startTime,
				address,
				sni,
				logger,
			)
		} else {
			res = handshakeWithEch(
				ctx,
				trace,
				conn,
				startTime,
				address,
				outerSni,
				logger,
			)
			// We need to set this explicitly because otherwise it will get
			// overridden with the outerSni in the case of ECH
			res.ServerName = sni
		}
		channel <- *res
	}()

	return channel, nil
}

func handshake(ctx context.Context, trace *measurexlite.Trace, conn net.Conn, zeroTime time.Time,
	address string, sni string, logger model.Logger) *model.ArchivalTLSOrQUICHandshakeResult {
	return doHandshake(ctx, trace, conn, zeroTime, address, sni, []byte{}, logger)
}

func handshakeWithEch(ctx context.Context, trace *measurexlite.Trace, conn net.Conn, zeroTime time.Time,
	address string, sni string, logger model.Logger) *model.ArchivalTLSOrQUICHandshakeResult {

	hs := doHandshake(ctx, trace, conn, zeroTime, address, sni, []byte("ECHCONFIG"), logger)
	hs.ECHConfig = "GREASE"
	hs.OuterServerName = sni
	return hs
}

func handshakeMaybePrintWithECH(doprint bool) string {
	if doprint {
		return "WithECH"
	}
	return ""
}

func doHandshake(ctx context.Context, trace *measurexlite.Trace, conn net.Conn, zeroTime time.Time, address string, sni string,
	echConfig []byte, logger model.Logger) *model.ArchivalTLSOrQUICHandshakeResult {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	tlsConfig := genTLSConfig(sni)
	if len(echConfig) > 0 {
		tlsConfig.EncryptedClientHelloConfigList = echConfig
	}

	tracedHandshaker := trace.NewTLSHandshakerStdlib(logger)

	ol := logx.NewOperationLogger(logger, "echcheck: TLSHandshake with ECH")
	start := time.Now()
	maybeTLSConn, err := tracedHandshaker.Handshake(ctx, conn, tlsConfig)
	finish := time.Now()
	ol.Stop(err)

	connState := netxlite.MaybeTLSConnectionState(maybeTLSConn)
	return measurexlite.NewArchivalTLSOrQUICHandshakeResult(0, start.Sub(zeroTime), "tcp", address, tlsConfig,
		connState, err, finish.Sub(zeroTime))
}

// We are creating the pool just once because there is a performance penalty
// when creating it every time. See https://github.com/ooni/probe/issues/2413.
var certpool = netxlite.NewMozillaCertPool()

// genTLSConfig generates tls.Config from a given SNI
func genTLSConfig(sni string) *tls.Config {
	return &tls.Config{ // #nosec G402 - we need to use a large TLS versions range for measuring
		RootCAs:            certpool,
		ServerName:         sni,
		NextProtos:         []string{"h2", "http/1.1"},
		InsecureSkipVerify: true, // #nosec G402 - it's fine to skip verify in a nettest
	}
}
