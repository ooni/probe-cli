// Package netxlogger is a logger for netx events.
//
// This package is a fork of github.com/ooni/netx/x/logger where
// we applied ooni/probe-engine specific customisations.
package netxlogger

import (
	"net/http"
	"strings"

	"github.com/ooni/probe-cli/v3/internal/engine/legacy/netx/modelx"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/tlsx"
)

// Logger is the interface we expect from a logger
type Logger interface {
	Debug(msg string)
	Debugf(format string, v ...interface{})
}

// Handler is a handler that logs events.
type Handler struct {
	logger Logger
}

// NewHandler returns a new logging handler.
func NewHandler(logger Logger) *Handler {
	return &Handler{logger: logger}
}

// OnMeasurement logs the specific measurement
func (h *Handler) OnMeasurement(m modelx.Measurement) {
	// DNS
	if m.ResolveStart != nil {
		h.logger.Debugf(
			"[httpTxID: %d] resolving: %s",
			m.ResolveStart.TransactionID,
			m.ResolveStart.Hostname,
		)
	}
	if m.ResolveDone != nil {
		h.logger.Debugf(
			"[httpTxID: %d] resolve done: %s, %s",
			m.ResolveDone.TransactionID,
			fmtError(m.ResolveDone.Error),
			m.ResolveDone.Addresses,
		)
	}

	// Syscalls
	if m.Connect != nil {
		h.logger.Debugf(
			"[httpTxID: %d] connect done: %s, %s (rtt=%s)",
			m.Connect.TransactionID,
			fmtError(m.Connect.Error),
			m.Connect.RemoteAddress,
			m.Connect.SyscallDuration,
		)
	}

	// TLS
	if m.TLSHandshakeStart != nil {
		h.logger.Debugf(
			"[httpTxID: %d] TLS handshake: (forceSNI='%s')",
			m.TLSHandshakeStart.TransactionID,
			m.TLSHandshakeStart.SNI,
		)
	}
	if m.TLSHandshakeDone != nil {
		h.logger.Debugf(
			"[httpTxID: %d] TLS done: %s, %s (alpn='%s')",
			m.TLSHandshakeDone.TransactionID,
			fmtError(m.TLSHandshakeDone.Error),
			tlsx.VersionString(m.TLSHandshakeDone.ConnectionState.Version),
			m.TLSHandshakeDone.ConnectionState.NegotiatedProtocol,
		)
	}

	// HTTP round trip
	if m.HTTPRequestHeadersDone != nil {
		proto := "HTTP/1.1"
		for key := range m.HTTPRequestHeadersDone.Headers {
			if strings.HasPrefix(key, ":") {
				proto = "HTTP/2.0"
				break
			}
		}
		h.logger.Debugf(
			"[httpTxID: %d] > %s %s %s",
			m.HTTPRequestHeadersDone.TransactionID,
			m.HTTPRequestHeadersDone.Method,
			m.HTTPRequestHeadersDone.URL.RequestURI(),
			proto,
		)
		if proto == "HTTP/2.0" {
			h.logger.Debugf(
				"[httpTxID: %d] > Host: %s",
				m.HTTPRequestHeadersDone.TransactionID,
				m.HTTPRequestHeadersDone.URL.Host,
			)
		}
		for key, values := range m.HTTPRequestHeadersDone.Headers {
			if strings.HasPrefix(key, ":") {
				continue
			}
			for _, value := range values {
				h.logger.Debugf(
					"[httpTxID: %d] > %s: %s",
					m.HTTPRequestHeadersDone.TransactionID,
					key, value,
				)
			}
		}
		h.logger.Debugf(
			"[httpTxID: %d] >", m.HTTPRequestHeadersDone.TransactionID)
	}
	if m.HTTPRequestDone != nil {
		h.logger.Debugf(
			"[httpTxID: %d] request sent; waiting for response",
			m.HTTPRequestDone.TransactionID,
		)
	}
	if m.HTTPResponseStart != nil {
		h.logger.Debugf(
			"[httpTxID: %d] start receiving response",
			m.HTTPResponseStart.TransactionID,
		)
	}
	if m.HTTPRoundTripDone != nil && m.HTTPRoundTripDone.Error == nil {
		h.logger.Debugf(
			"[httpTxID: %d] < %s %d %s",
			m.HTTPRoundTripDone.TransactionID,
			m.HTTPRoundTripDone.ResponseProto,
			m.HTTPRoundTripDone.ResponseStatusCode,
			http.StatusText(int(m.HTTPRoundTripDone.ResponseStatusCode)),
		)
		for key, values := range m.HTTPRoundTripDone.ResponseHeaders {
			for _, value := range values {
				h.logger.Debugf(
					"[httpTxID: %d] < %s: %s",
					m.HTTPRoundTripDone.TransactionID,
					key, value,
				)
			}
		}
		h.logger.Debugf(
			"[httpTxID: %d] <", m.HTTPRoundTripDone.TransactionID)
	}

	// HTTP response body
	if m.HTTPResponseBodyPart != nil {
		h.logger.Debugf(
			"[httpTxID: %d] body part: %s, %d",
			m.HTTPResponseBodyPart.TransactionID,
			fmtError(m.HTTPResponseBodyPart.Error),
			len(m.HTTPResponseBodyPart.Data),
		)
	}
	if m.HTTPResponseDone != nil {
		h.logger.Debugf(
			"[httpTxID: %d] end of response",
			m.HTTPResponseDone.TransactionID,
		)
	}
}

func fmtError(err error) (s string) {
	s = "success"
	if err != nil {
		s = err.Error()
	}
	return
}
