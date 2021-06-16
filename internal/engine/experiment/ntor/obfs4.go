package ntor

import (
	"context"
	"net"

	"github.com/ooni/probe-cli/v3/internal/engine/netx/mockablex"
	"github.com/ooni/probe-cli/v3/internal/ptx"
)

// doOBFS4 performs an OBFS4 handshake with "obfs4" types.
func (svc *service) doOBFS4(ctx context.Context, out *serviceOutput, conn net.Conn) {
	defer conn.Close() // we own it
	ob4dialer := &ptx.OBFS4Dialer{
		Address:     out.results.TargetAddress,
		Cert:        out.ptCert(),
		DataDir:     "", // TODO(bassosimone): figure out datadir
		Fingerprint: out.ptFingerprint(),
		IATMode:     out.ptIATMode(),
		UnderlyingDialer: &mockablex.Dialer{
			MockDialContext: func(ctx context.Context, network string, address string) (net.Conn, error) {
				return conn, nil
			},
		},
	}
	// TODO(bassosimone): implement error wrapping here?
	ob4conn, err := ob4dialer.DialContext(ctx)
	if err != nil {
		out.err = err
		out.operation = "obfs4_handshake"
		return
	}
	ob4conn.Close()
}

// ptCert returns the certificate for obfs4
func (out *serviceOutput) ptCert() string {
	v, ok := out.in.target.Params["cert"]
	if !ok || len(v) < 1 {
		return ""
	}
	return v[0]
}

// ptFingerprint returns the fingerprint for obfs4
func (out *serviceOutput) ptFingerprint() string {
	// TODO(bassosimone): how to get the fingerprint?
	return ""
}

// ptIATMode returns the IAT mode for obfs4
func (out *serviceOutput) ptIATMode() string {
	v, ok := out.in.target.Params["iat-mode"]
	if !ok || len(v) < 1 {
		return ""
	}
	return v[0]
}
