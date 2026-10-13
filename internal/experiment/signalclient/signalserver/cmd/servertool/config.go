package main

//
// config: Signal-Server's test configuration (service/src/test/resources/config/test.yml) with a
// TLS connector for chat next to the original h2c one, and the gRPC front end with TLS. Each edit
// replaces a known excerpt, and fails if the excerpt changed, rather than parsing the YAML.
//

import (
	"fmt"
	"os"
	"strings"
)

const configChanged = "test.yml changed: update signalserver/run.sh"

func writeTestConfig(src, dst, keyStorePassword string) error {
	data, err := os.ReadFile(src) // #nosec G304 -- a path the harness passes
	if err != nil {
		return err
	}
	s := string(data)

	old := "  applicationConnectors:\n    - type: h2c\n      port: 8080\n"
	if !strings.Contains(s, old) {
		return fmt.Errorf("%s", configChanged)
	}
	quoted := "'" + strings.ReplaceAll(keyStorePassword, "'", "''") + "'" // a YAML single-quoted string
	tls := "      keyStoreType: PKCS12\n      keyStorePassword: " + quoted + "\n"
	// Settings measured on production with tlsprofile.sh (2026-10). Dropwizard's h2 connector
	// requires TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256 in its list.
	chat := "    - type: h2\n      port: 8443\n      keyStorePath: /certs/chat.p12\n" + tls +
		"      supportedProtocols: [TLSv1.3, TLSv1.2]\n" +
		"      supportedCipherSuites: [TLS_AES_256_GCM_SHA384, TLS_CHACHA20_POLY1305_SHA256, TLS_AES_128_GCM_SHA256," +
		" TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384, TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256," +
		" TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384, TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256," +
		" TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA384, TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA384," +
		" TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256, TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256]\n"
	s = strings.ReplaceAll(s, old, old+chat) // the h2c connector stays: the gRPC front end proxies websockets to it

	// The gRPC front end takes its certificates from tlsKeyStore, picking them by SNI from their SANs.
	old = "tlsKeyStore:\n"
	if strings.Count(s, old) != 1 {
		return fmt.Errorf("%s", configChanged)
	}
	s = strings.Replace(s, old, old+"  path: /certs/grpc.p12\n", 1)

	old = "grpc:\n  port: 50051\n  websocketPort: 8080\n  h2c: true\n"
	if !strings.Contains(s, old) {
		return fmt.Errorf("%s", configChanged)
	}
	s = strings.ReplaceAll(s, old, "grpc:\n  bindAddress: 0.0.0.0\n  port: 8444\n  websocketPort: 8080\n  h2c: false\n")
	return os.WriteFile(dst, []byte(s), 0o600)
}
