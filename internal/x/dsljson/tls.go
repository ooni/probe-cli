package dsljson

import (
	"crypto/x509"
	"encoding/json"
	"errors"

	"github.com/ooni/probe-cli/v3/internal/x/dslvm"
)

type tlsHandshakeValue struct {
	Input              string   `json:"input"`
	InsecureSkipVerify bool     `json:"insecure_skip_verify"`
	NextProtos         []string `json:"next_protos"`
	Output             string   `json:"output"`
	RootCAs            []string `json:"root_cas"`
	ServerName         string   `json:"server_name"`
}

func (lx *loader) onTLSHandshake(raw json.RawMessage) error {
	// parse the raw value
	var value tlsHandshakeValue
	if err := json.Unmarshal(raw, &value); err != nil {
		return err
	}

	// create the required output registers
	output, err := registerMakeOutput[*dslvm.TLSConnection](lx, value.Output)
	if err != nil {
		return err
	}

	// fetch the required input register
	input, err := registerPopInput[*dslvm.TCPConnection](lx, value.Input)
	if err != nil {
		return err
	}

	// create the X509 cert pool
	var pool *x509.CertPool
	for _, cert := range value.RootCAs {
		if pool == nil {
			pool = x509.NewCertPool()
		}
		if !pool.AppendCertsFromPEM([]byte(cert)) {
			return errors.New("cannot add PEM-encoded cert to X509 cert pool")
		}
	}

	// instantiate the ASM stage
	sx := &dslvm.TLSHandshakeStage{
		Input:              input,
		InsecureSkipVerify: value.InsecureSkipVerify,
		NextProtos:         value.NextProtos,
		Output:             output,
		RootCAs:            pool,
		ServerName:         value.ServerName,
	}

	// remember the stage for later
	lx.stages = append(lx.stages, sx)
	return nil
}
