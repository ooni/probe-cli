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

type EchMode int

const (
	NoECH EchMode = iota
	GreaseECH
	RealECH
)

// Connect to `host` at `address` and attempt a TLS handshake.  When using real ECH, `ecl` must
// contain the ECHConfigList to use; in other modes, `ecl` is ignored. If the ECH config provides
// a different OuterServerName than `host`, it will be recorded in the result and used per go's
// tls package's behavior.
// Returns a channel that will contain the archival result of the handshake.
func connectAndHandshake(ctx context.Context, mode EchMode, ecl echConfigList, startTime time.Time,
	address string, host string, logger model.Logger) (chan model.ArchivalTLSOrQUICHandshakeResult, error) {

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
		switch mode {
		case NoECH:
			res = handshakeWithoutEch(ctx, conn, startTime, address, host, logger)
		case GreaseECH:
			res = handshakeWithGreaseyEch(ctx, conn, startTime, address, host, logger)
		case RealECH:
			res = handshakeWithRealEch(ctx, conn, startTime, address, host, ecl, logger)
		}
		channel <- *res
	}()

	return channel, nil
}

func handshakeWithoutEch(ctx context.Context, conn net.Conn, zeroTime time.Time,
	address string, sni string, logger model.Logger) *model.ArchivalTLSOrQUICHandshakeResult {
	return handshakeWithExtension(ctx, conn, zeroTime, address, sni, []utls.TLSExtension{}, logger)
}

func handshakeWithGreaseyEch(ctx context.Context, conn net.Conn, zeroTime time.Time,
	address string, sni string, logger model.Logger) *model.ArchivalTLSOrQUICHandshakeResult {
	payload, err := generateGreaseExtension(rand.Reader)
	if err != nil {
		panic("failed to generate grease ECH: " + err.Error())
	}

	var utlsEchExtension utls.GenericExtension

	utlsEchExtension.Id = echExtensionType
	utlsEchExtension.Data = payload

	hs := handshakeWithExtension(ctx, conn, zeroTime, address, sni, []utls.TLSExtension{&utlsEchExtension}, logger)
	hs.ECHConfig = "GREASE"
	hs.OuterServerName = sni
	return hs
}

func handshakeMaybePrintWithECH(doprint bool) string {
	if doprint {
		return "WithGreaseECH"
	}
	return ""
}

// ECHConfigList must be valid, non-empty, and to specify only one unique PublicName,
// i.e. OuterServerName across all configs.  Host is the service to connect to, i.e.
// the inner SNI.
func handshakeWithRealEch(ctx context.Context, conn net.Conn, zeroTime time.Time,
	address string, host string, ecl echConfigList, logger model.Logger) *model.ArchivalTLSOrQUICHandshakeResult {

	tlsConfig := genEchTLSConfig(host, ecl)

	ol := logx.NewOperationLogger(logger, "echcheck: TLSHandshakeWithRealECH")
	start := time.Now()
	maybeTLSConn, err := tls.Dial("tcp", address, tlsConfig)
	finish := time.Now()
	ol.Stop(err)

	connState := netxlite.MaybeTLSConnectionState(maybeTLSConn)
	hs := measurexlite.NewArchivalTLSOrQUICHandshakeResult(0, start.Sub(zeroTime), "tcp", address, tlsConfig,
		connState, err, finish.Sub(zeroTime))
	hs.ECHConfig = ecl.Base64()
	hs.OuterServerName = string(ecl.Configs[0].PublicName)
	return hs
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
		InsecureSkipVerify: true, // #nosec G402 - it's fine to skip verify in a nettest
	}
}

func genEchTLSConfig(host string, ecl echConfigList) *tls.Config {
	return &tls.Config{
		EncryptedClientHelloConfigList: ecl.raw,
		// This will be used as the inner SNI and we will validate
		// we get a certificate for this name.  The outer SNI will
		// be set based on the ECH config.
		ServerName: host,
	}
}
