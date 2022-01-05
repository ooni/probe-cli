package measurex

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"time"

	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/ptx"
)

//
// API for reducing boilerplate for simple measurements.
//

// EasyHTTPGET performs a GET with the given URL and default headers.
//
// Arguments:
//
// - ctx is the context for deadline/timeout/cancellation;
//
// - timeout is the timeout for the whole operation;
//
// - URL is the URL to GET;
//
// Returns:
//
// - meas is a JSON serializable OONI measurement (this
// field will never be a nil pointer);
//
// - failure is either nil or a pointer to a OONI failure.
func (mx *Measurer) EasyHTTPGET(ctx context.Context, timeout time.Duration,
	URL string) (meas *ArchivalMeasurement, failure *string) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	db := &MeasurementDB{}
	req, err := NewHTTPRequestWithContext(ctx, "GET", URL, nil)
	if err != nil {
		failure := err.Error()
		return NewArchivalMeasurement(db.AsMeasurement()), &failure
	}
	txp := mx.NewTracingHTTPTransportWithDefaultSettings(db)
	resp, err := txp.RoundTrip(req)
	if err != nil {
		failure := err.Error()
		return NewArchivalMeasurement(db.AsMeasurement()), &failure
	}
	resp.Body.Close()
	return NewArchivalMeasurement(db.AsMeasurement()), nil
}

// EasyTLSConfig helps you to generate a *tls.Config.
type EasyTLSConfig struct {
	config *tls.Config
}

// NewEasyTLSConfig creates a new EasyTLSConfig instance.
func NewEasyTLSConfig() *EasyTLSConfig {
	return &EasyTLSConfig{config: &tls.Config{}}
}

// NewEasyTLSConfigWithServerName creates a new EasyTLSConfig
// with an already configured value for ServerName.
func NewEasyTLSConfigWithServerName(serverName string) *EasyTLSConfig {
	return NewEasyTLSConfig().ServerName(serverName)
}

// ServerName sets the SNI value.
func (easy *EasyTLSConfig) ServerName(v string) *EasyTLSConfig {
	easy.config.ServerName = v
	return easy
}

// InsecureSkipVerify disables TLS verification.
func (easy *EasyTLSConfig) InsecureSkipVerify(v bool) *EasyTLSConfig {
	easy.config.InsecureSkipVerify = v
	return easy
}

// RootCAs allows the set the CA pool.
func (easy *EasyTLSConfig) RootCAs(v *x509.CertPool) *EasyTLSConfig {
	easy.config.RootCAs = v
	return easy
}

// asTLSConfig converts an *EasyTLSConfig to a *tls.Config.
func (easy *EasyTLSConfig) asTLSConfig() *tls.Config {
	if easy == nil || easy.config == nil {
		return &tls.Config{}
	}
	return easy.config
}

// EasyTLSConnectAndHandshake performs a TCP connect to a TCP endpoint
// followed by a TLS handshake using the given config.
//
// Arguments:
//
// - ctx is the context for deadline/timeout/cancellation;
//
// - endpoint is the TCP endpoint to connect to (e.g.,
// 8.8.8.8:443 where the address part of the endpoint MUST
// be an IPv4 or IPv6 address and MUST NOT be a domain);
//
// - tlsConfig is the EasyTLSConfig to use (MUST NOT be nil).
//
// Returns:
//
// - meas is a JSON serializable OONI measurement (this
// field will never be a nil pointer);
//
// - failure is either nil or a pointer to a OONI failure.
//
// Note:
//
// - we use the Measurer's TCPConnectTimeout and TLSHandshakeTimeout.
func (mx *Measurer) EasyTLSConnectAndHandshake(ctx context.Context, endpoint string,
	tlsConfig *EasyTLSConfig) (meas *ArchivalMeasurement, failure *string) {
	db := &MeasurementDB{}
	conn, err := mx.TLSConnectAndHandshakeWithDB(ctx, db, endpoint, tlsConfig.asTLSConfig())
	if err != nil {
		failure := err.Error()
		return NewArchivalMeasurement(db.AsMeasurement()), &failure
	}
	conn.Close()
	return NewArchivalMeasurement(db.AsMeasurement()), nil
}

// EasyTCPConnect performs a TCP connect to a TCP endpoint.
//
// Arguments:
//
// - ctx is the context for deadline/timeout/cancellation;
//
// - endpoint is the TCP endpoint to connect to (e.g.,
// 8.8.8.8:443 where the address part of the endpoint MUST
// be an IPv4 or IPv6 address and MUST NOT be a domain).
//
// Returns:
//
// - meas is a JSON serializable OONI measurement (this
// field will never be a nil pointer);
//
// - failure is either nil or a pointer to a OONI failure.
//
// Note:
//
// - we use the Measurer's TCPConnectTimeout.
func (mx *Measurer) EasyTCPConnect(ctx context.Context,
	endpoint string) (meas *ArchivalMeasurement, failure *string) {
	// Note: TCPConnectWithDB has a default timeout.
	db := &MeasurementDB{}
	conn, err := mx.TCPConnectWithDB(ctx, db, endpoint)
	if err != nil {
		failure := err.Error()
		return NewArchivalMeasurement(db.AsMeasurement()), &failure
	}
	conn.Close()
	return NewArchivalMeasurement(db.AsMeasurement()), nil
}

// easyOBFS4Params contains params for OBFS4.
type easyOBFS4Params struct {
	// Cert contains the MANDATORY certificate parameter.
	Cert string

	// DataDir is the MANDATORY directory where to store obfs4 data.
	DataDir string

	// Fingerprint is the MANDATORY bridge fingerprint.
	Fingerprint string

	// IATMode contains the MANDATORY iat-mode parameter.
	IATMode string
}

// newEasyOBFS4Params constructs an EasyOBFS4Params structure
// from the map[string][]string returned by the OONI API.
//
// This function will only fail when the rawParams contains
// more than one entry for each input key.
func newEasyOBFS4Params(dataDir string, rawParams map[string][]string) (*easyOBFS4Params, error) {
	out := &easyOBFS4Params{DataDir: dataDir}
	for key, values := range rawParams {
		var field *string
		switch key {
		case "cert":
			field = &out.Cert
		case "fingerprint":
			field = &out.Fingerprint
		case "iat-mode":
			field = &out.IATMode
		default:
			continue // not interested
		}
		if len(values) != 1 {
			return nil, fmt.Errorf("obfs4: expected exactly one value for %s", key)
		}
		*field = values[0]
	}
	// Assume that the API knows what it's returning, so don't bother
	// checking whether some fields are missing. If this happens, it
	// will be the obfs4 library task to tell us about that.
	return out, nil
}

// EasyOBFS4ConnectAndHandshake performs a TCP connect to a TCP endpoint
// followed by an OBFS4 handshake. This function is designed to receive
// in input the Tor bridges from the OONI API.
//
// Arguments:
//
// - ctx is the context for deadline/timeout/cancellation;
//
// - timeout is the timeout for the whole operation;
//
// - endpoint is the TCP endpoint to connect to (e.g.,
// 8.8.8.8:443 where the address part of the endpoint MUST
// be an IPv4 or IPv6 address and MUST NOT be a domain);
//
// - dataDir is the data directory to use for obfs4;
//
// - rawParams contains raw obfs4 params from the OONI API.
//
// Returns:
//
// - meas is a JSON serializable OONI measurement (this
// field will never be a nil pointer);
//
// - failure is either nil or a pointer to a OONI failure.
func (mx *Measurer) EasyOBFS4ConnectAndHandshake(ctx context.Context,
	timeout time.Duration, endpoint string, dataDir string,
	rawParams map[string][]string) (meas *ArchivalMeasurement, failure *string) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	db := &MeasurementDB{}
	params, err := newEasyOBFS4Params(dataDir, rawParams)
	if err != nil {
		failure := err.Error()
		return NewArchivalMeasurement(db.AsMeasurement()), &failure
	}
	conn, err := mx.TCPConnectWithDB(ctx, db, endpoint)
	if err != nil {
		failure := err.Error()
		return NewArchivalMeasurement(db.AsMeasurement()), &failure
	}
	defer conn.Close()
	dialer := netxlite.NewSingleUseDialer(conn)
	obfs4 := ptx.OBFS4Dialer{
		Address:          endpoint,
		Cert:             params.Cert,
		DataDir:          params.DataDir,
		Fingerprint:      params.Fingerprint,
		IATMode:          params.IATMode,
		UnderlyingDialer: dialer,
	}
	o4conn, err := obfs4.DialContext(ctx)
	if err != nil {
		failure := err.Error()
		return NewArchivalMeasurement(db.AsMeasurement()), &failure
	}
	o4conn.Close()
	return NewArchivalMeasurement(db.AsMeasurement()), nil
}
