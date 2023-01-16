package echcheck

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	utls "gitlab.com/yawning/utls.git"
	"net"
	"time"
)

const echExtensionType uint16 = 0xfe0d

func handshake(ctx context.Context, conn net.Conn, zeroTime time.Time, address string, sni string) *model.ArchivalTLSOrQUICHandshakeResult {
	return handshakeWithExtension(ctx, conn, zeroTime, address, sni, []utls.TLSExtension{})
}

func handshakeWithEch(ctx context.Context, conn net.Conn, zeroTime time.Time, address string, sni string) *model.ArchivalTLSOrQUICHandshakeResult {
	payload, err := generateGreaseExtension(rand.Reader)
	if err != nil {
		panic("failed to generate grease ECH: " + err.Error())
	}

	var utlsEchExtension utls.GenericExtension

	utlsEchExtension.Id = echExtensionType
	utlsEchExtension.Data = payload

	return handshakeWithExtension(ctx, conn, zeroTime, address, sni, []utls.TLSExtension{&utlsEchExtension})
}

func handshakeWithExtension(ctx context.Context, conn net.Conn, zeroTime time.Time, address string, sni string, extensions []utls.TLSExtension) *model.ArchivalTLSOrQUICHandshakeResult {
	tlsConfig := genTLSConfig(sni)

	handshakerConstructor := newHandshakerWithExtensions(extensions)
	tracedHandshaker := handshakerConstructor(log.Log, &utls.HelloFirefox_Auto)

	start := time.Now()
	_, connState, err := tracedHandshaker.Handshake(ctx, conn, tlsConfig)
	finish := time.Now()

	return measurexlite.NewArchivalTLSOrQUICHandshakeResult(0, start.Sub(zeroTime), "tcp", address, tlsConfig,
		connState, err, finish.Sub(zeroTime))
}

// genTLSConfig generates tls.Config from a given SNI
func genTLSConfig(sni string) *tls.Config {
	return &tls.Config{
		RootCAs:            netxlite.NewDefaultCertPool(),
		ServerName:         sni,
		NextProtos:         []string{"h2", "http/1.1"},
		InsecureSkipVerify: true,
	}
}
