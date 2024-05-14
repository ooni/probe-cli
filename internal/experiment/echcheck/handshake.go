package echcheck

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"net"
	"time"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/logx"
	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	utls "gitlab.com/yawning/utls.git"
)

const echExtensionType uint16 = 0xfe0d

func handshake(ctx context.Context, conn net.Conn, zeroTime time.Time,
	address string, sni string, logger model.Logger) *model.ArchivalTLSOrQUICHandshakeResult {
	return handshakeWithExtension(ctx, conn, zeroTime, address, sni, []utls.TLSExtension{}, logger)
}

func handshakeWithEch(ctx context.Context, conn net.Conn, zeroTime time.Time,
	address string, sni string, logger model.Logger) *model.ArchivalTLSOrQUICHandshakeResult {
	payload, err := generateGreaseExtension(rand.Reader)
	if err != nil {
		panic("failed to generate grease ECH: " + err.Error())
	}

	var utlsEchExtension utls.GenericExtension

	utlsEchExtension.Id = echExtensionType
	utlsEchExtension.Data = payload

	return handshakeWithExtension(ctx, conn, zeroTime, address, sni, []utls.TLSExtension{&utlsEchExtension}, logger)
}

func handshakeMaybePrintWithECH(doprint bool) string {
	if doprint {
		return "WithECH"
	}
	return ""
}

func handshakeWithExtension(ctx context.Context, conn net.Conn, zeroTime time.Time, address string, sni string,
	extensions []utls.TLSExtension, logger model.Logger) *model.ArchivalTLSOrQUICHandshakeResult {
	tlsConfig := genTLSConfig(sni)

	handshakerConstructor := newHandshakerWithExtensions(extensions)
	tracedHandshaker := handshakerConstructor(log.Log, &utls.HelloFirefox_Auto)

	ol := logx.NewOperationLogger(logger, "echcheck: TLSHandshake%s", handshakeMaybePrintWithECH(len(extensions) > 0))
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
		InsecureSkipVerify: true,
	}
}
