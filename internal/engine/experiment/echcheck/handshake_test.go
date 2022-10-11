package echcheck

import (
	"context"
	"net"
	"testing"
	"time"

	utls "gitlab.com/yawning/utls.git"
)

// To generate run: openssl ecparam -genkey -name secp384r1 -out server.key
const serverKey = `-----BEGIN EC PARAMETERS-----
BgUrgQQAIg==
-----END EC PARAMETERS-----
-----BEGIN EC PRIVATE KEY-----
MIGkAgEBBDBrz5yMcQmeQl0rf1TFfXgYS69L9KE3RzVr8GKLODQWJcNG2K1hCT4x
CjTapWWvz/OgBwYFK4EEACKhZANiAAToZ4BH+ZnBcafzSwNQwFlKheonY4I+kL61
m5nkUfPnlPJXjM5IEzLXaeMAEtDRJJ7TcqLNzYW3yEiO4wfLZZvNJoieql8wtem3
g+ffq7audAAx83okFBADqlZjSGF4HzI=
-----END EC PRIVATE KEY-----`

// To generate run: openssl req -new -x509 -sha256 -key server.key -out server.crt -days 8300
const serverCertificate = `-----BEGIN CERTIFICATE-----
MIICKjCCAbECCQCxZjKxtpVMuzAKBggqhkjOPQQDAjB/MQswCQYDVQQGEwJJTjES
MBAGA1UECAwJS2FybmF0YWthMRIwEAYDVQQHDAlCZW5nYWx1cnUxCzAJBgNVBAoM
Ak5BMQswCQYDVQQLDAJOQTENMAsGA1UEAwwEdGVzdDEfMB0GCSqGSIb3DQEJARYQ
dGVzdEBleGFtcGxlLm9yZzAeFw0yMjEwMDExMjE5MzRaFw00NTA2MjIxMjE5MzRa
MH8xCzAJBgNVBAYTAklOMRIwEAYDVQQIDAlLYXJuYXRha2ExEjAQBgNVBAcMCUJl
bmdhbHVydTELMAkGA1UECgwCTkExCzAJBgNVBAsMAk5BMQ0wCwYDVQQDDAR0ZXN0
MR8wHQYJKoZIhvcNAQkBFhB0ZXN0QGV4YW1wbGUub3JnMHYwEAYHKoZIzj0CAQYF
K4EEACIDYgAE6GeAR/mZwXGn80sDUMBZSoXqJ2OCPpC+tZuZ5FHz55TyV4zOSBMy
12njABLQ0SSe03Kizc2Ft8hIjuMHy2WbzSaInqpfMLXpt4Pn36u2rnQAMfN6JBQQ
A6pWY0hheB8yMAoGCCqGSM49BAMCA2cAMGQCMAs9e6y7257CxQjD4WASxRQfU0Zl
LYnocU7qH90HM8AdSQ/5pEx5H/1uFXEnw6pS1gIwIDIPMmQrlKJni4rbAI6zjh40
D49LMqpzQNG4Td4xDdrJT1HRRyhCOTCrr6t344AA
-----END CERTIFICATE-----`

func TestHandshake(t *testing.T) {
	go listenAndServe(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var dialer net.Dialer
	conn, err := dialer.DialContext(ctx, "tcp", "localhost:443")
	if err != nil {
		t.Fatal(err)
	}

	result := handshakeWithEch(ctx, conn, time.Now(), "0.0.0.0:0", "example.org")
	if result == nil {
		t.Fatal("expected result")
	}

	if result.SoError != nil {
		t.Fatal("did not expect error, got: ", result.SoError)
	}

	if result.Failure != nil {
		t.Fatal("did not expect error, got: ", *result.Failure)
	}
}

func listenAndServe(t *testing.T) {
	cer, err := utls.X509KeyPair([]byte(serverCertificate), []byte(serverKey))
	if err != nil {
		panic(err)
	}

	config := &utls.Config{Certificates: []utls.Certificate{cer}}
	ln, err := utls.Listen("tcp", ":443", config)
	if err != nil {
		panic(err)
	}
	defer ln.Close()

	for {
		conn, err := ln.Accept()
		if err != nil {
			panic(err)
		}

		defer conn.Close()

		_, err = conn.Write([]byte("success\n"))
		if err != nil {
			panic(err)
		}

		break
	}
	return
}
