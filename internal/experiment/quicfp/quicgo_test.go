package quicfp

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/quic-go/quic-go"
)

// TestRunAgainstQUICGo runs the experiment against a quic-go server on
// loopback, which speaks v1 and v2. Unlike the round-trip tests, which decrypt
// packets with this package's own code, this shows that a real QUIC stack can
// decrypt them: it answers a version it speaks with an Initial (a TLS alert,
// as the synthetic ClientHello has no quic_transport_parameters), any other
// version with Version Negotiation, and drops a datagram under 1200 bytes, as
// RFC 9000 §14.1 requires.
func TestRunAgainstQUICGo(t *testing.T) {
	addr := startQUICGoServer(t)
	meas := &model.Measurement{MeasurementStartTimeSaved: time.Now()}
	args := &model.ExperimentArgs{
		Measurement: meas,
		Session:     &mocks.Session{MockLogger: func() model.Logger { return model.DiscardLogger }},
		Target:      &Target{Config: &Config{VectorTimeout: 500}, URL: addr.String()},
	}
	if err := NewExperimentMeasurer().Run(context.Background(), args); err != nil {
		t.Fatal(err)
	}
	for _, vr := range meas.TestKeys.(*TestKeys).Vectors {
		want := "initial"
		switch {
		case vr.Name == "quic_feature_below_1200_bytes":
			want = "" // no answer
		case vr.Name == "quic_feature_forced_negotiation",
			vr.Category == CategoryQUICVersion && *vr.QUICVersion != 1 && *vr.QUICVersion != versionV2:
			want = "version_negotiation"
		}
		p := vr.Pings[0]
		got := ""
		if !p.Blocked {
			got = p.Responses[len(p.Responses)-1].Type
		}
		if got != want {
			t.Errorf("%s: got answer %q, want %q", vr.Name, got, want)
		}
	}
}

// startQUICGoServer starts a quic-go server for example.com on a loopback
// port and returns its address.
func startQUICGoServer(t *testing.T) *net.UDPAddr {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "example.com"},
		DNSNames:     []string{"example.com"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{{Certificate: [][]byte{der}, PrivateKey: key}},
		NextProtos:   []string{"h3"},
	}
	ln, err := quic.ListenAddr("127.0.0.1:0", tlsConfig, &quic.Config{
		Versions: []quic.Version{quic.Version1, quic.Version2},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	go func() {
		for {
			if _, err := ln.Accept(context.Background()); err != nil {
				return
			}
		}
	}()
	return ln.Addr().(*net.UDPAddr)
}
