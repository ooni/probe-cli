package echcheck

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"net/url"
	"time"

	"github.com/ooni/probe-cli/v3/internal/logx"
	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// We can't see which outerservername go std lib selects, so we preemptively
// make sure it's unambiguous for a given ECH Config List.  If the ecl is
// empty, return an empty string.
func getUnambiguousOuterServerName(ecl []byte) (string, error) {
	if len(ecl) == 0 {
		return "", nil
	}
	configs, err := parseECHConfigList(ecl)
	if err != nil {
		return "", fmt.Errorf("failed to parse ECH config: %w", err)
	}
	outerServerName := string(configs[0].PublicName)
	for _, ec := range configs {
		if string(ec.PublicName) != outerServerName {
			// It's perfectly valid to have multiple ECH configs with different
			// `PublicName`s. But, since we can't see which one is selected by
			// go's tls package, we can't accurately record OuterServerName.
			return "", fmt.Errorf("ambigious OuterServerName for config")
		}
	}
	return outerServerName, nil
}

func startHandshake(ctx context.Context, echConfigList []byte, isGrease bool, startTime time.Time, address string, target *url.URL, logger model.Logger, testOnlyRootCAs *x509.CertPool) (chan TestKeys, error) {

	channel := make(chan TestKeys)

	tlsConfig := genEchTLSConfig(target.Hostname(), echConfigList, testOnlyRootCAs)

	go func() {
		tk := TestKeys{}
		tk = handshake(ctx, isGrease, startTime, address, logger, tlsConfig, tk)
		channel <- tk
	}()

	return channel, nil
}

// Add to tk TestKeys all events as they occur.  May call self recursively using retry_configs.
func handshake(ctx context.Context, isGrease bool, startTime time.Time, address string, logger model.Logger, tlsConfig *tls.Config, tk TestKeys) TestKeys {
	var d string
	if isGrease {
		d = " (GREASE)"
	} else if len(tlsConfig.EncryptedClientHelloConfigList) > 0 {
		d = " (RealECH)"
	}

	ol1 := logx.NewOperationLogger(logger, "echcheck: TCPConnect %s", address)
	trace := measurexlite.NewTrace(0, startTime)
	dialer := trace.NewDialerWithoutResolver(logger)
	conn, err := dialer.DialContext(ctx, "tcp", address)
	ol1.Stop(err)
	newTcpcs := trace.TCPConnects()
	tk.TCPConnects = append(tk.TCPConnects, newTcpcs...)

	ol2 := logx.NewOperationLogger(logger, "echcheck: DialTLS%s", d)
	start := time.Now()
	maybeTLSConn := tls.Client(conn, tlsConfig)
	err = maybeTLSConn.HandshakeContext(ctx)
	finish := time.Now()
	ol2.Stop(err)

	retryConfigs := []byte{}
	if echErr, ok := err.(*tls.ECHRejectionError); ok && isGrease {
		if len(echErr.RetryConfigList) > 0 {
			retryConfigs = echErr.RetryConfigList
		}
		// We ignore this error in crafting our TLSOrQUICHandshakeResult
		// since the *golang* error is expected and merely indicates we
		// didn't get the ECH setup we wanted.  It does NOT indicate that
		// that the handshake itself was a failure.
		// TODO: Can we *confirm* there wasn't a separate TLS failure?  This might be ambiguous :-(
		// TODO: Confirm above semantics with OONI team.
		err = nil
	}

	connState := netxlite.MaybeTLSConnectionState(maybeTLSConn)
	hs := measurexlite.NewArchivalTLSOrQUICHandshakeResult(0, start.Sub(startTime),
		"tcp", address, tlsConfig, connState, err, finish.Sub(startTime))
	if isGrease {
		hs.ECHConfig = "GREASE"
	} else {
		hs.ECHConfig = base64.StdEncoding.EncodeToString(tlsConfig.EncryptedClientHelloConfigList)
	}
	osn, err := getUnambiguousOuterServerName(tlsConfig.EncryptedClientHelloConfigList)
	if err != nil {
		msg := fmt.Sprintf("can't determine OuterServerName: %s", err)
		hs.SoError = &msg
	}
	hs.OuterServerName = osn
	tk.TLSHandshakes = append(tk.TLSHandshakes, hs)

	if len(retryConfigs) > 0 {
		tlsConfig.EncryptedClientHelloConfigList = retryConfigs
		tk = handshake(ctx, false, startTime, address, logger, tlsConfig, tk)
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
