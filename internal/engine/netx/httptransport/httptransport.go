// Package httptransport contains HTTP transport extensions.
package httptransport

import (
	"crypto/tls"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// Config contains the configuration required for constructing an HTTP transport
type Config struct {
	Dialer     model.Dialer
	QUICDialer model.QUICDialer
	TLSDialer  model.TLSDialer
	TLSConfig  *tls.Config
}
