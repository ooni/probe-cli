// Package netxlogger is a logger for netx events.
//
// This package is a fork of github.com/ooni/netx/x/logger where
// we applied ooni/probe-engine specific customisations.
package netxlogger

import (
	"net/http"
	"strings"

	"github.com/ooni/probe-cli/v3/internal/engine/legacy/netx/modelx"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
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
			"resolving: %s",
			m.ResolveStart.Hostname,
		)
	}
	if m.ResolveDone != nil {
		h.logger.Debugf(
			"resolve done: %s, %s",
			fmtError(m.ResolveDone.Error),
			m.ResolveDone.Addresses,
		)
	}

	// Syscalls
	if m.Connect != nil {
		h.logger.Debugf(
			"connect done: %s, %s (rtt=%s)",
			fmtError(m.Connect.Error),
			m.Connect.RemoteAddress,
			m.Connect.SyscallDuration,
		)
	}

	// TLS
	if m.TLSHandshakeStart != nil {
		h.logger.Debugf(
			"TLS handshake: (forceSNI='%s')",
			m.TLSHandshakeStart.SNI,
		)
	}
	if m.TLSHandshakeDone != nil {
		h.logger.Debugf(
			"TLS done: %s, %s (alpn='%s')",
			fmtError(m.TLSHandshakeDone.Error),
			netxlite.TLSVersionString(m.TLSHandshakeDone.ConnectionState.Version),
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
			"> %s %s %s",
			m.HTTPRequestHeadersDone.Method,
			m.HTTPRequestHeadersDone.URL.RequestURI(),
			proto,
		)
		if proto == "HTTP/2.0" {
			h.logger.Debugf(
				"> Host: %s",
				m.HTTPRequestHeadersDone.URL.Host,
			)
		}
		for key, values := range m.HTTPRequestHeadersDone.Headers {
			if strings.HasPrefix(key, ":") {
				continue
			}
			for _, value := range values {
				h.logger.Debugf(
					"> %s: %s",
					key, value,
				)
			}
		}
		h.logger.Debug(">")
	}
	if m.HTTPRequestDone != nil {
		h.logger.Debug("request sent; waiting for response")
	}
	if m.HTTPResponseStart != nil {
		h.logger.Debug("start receiving response")
	}
	if m.HTTPRoundTripDone != nil && m.HTTPRoundTripDone.Error == nil {
		h.logger.Debugf(
			"< %s %d %s",
			m.HTTPRoundTripDone.ResponseProto,
			m.HTTPRoundTripDone.ResponseStatusCode,
			http.StatusText(int(m.HTTPRoundTripDone.ResponseStatusCode)),
		)
		for key, values := range m.HTTPRoundTripDone.ResponseHeaders {
			for _, value := range values {
				h.logger.Debugf(
					"< %s: %s",
					key, value,
				)
			}
		}
		h.logger.Debug("<")
	}

	// HTTP response body
	if m.HTTPResponseBodyPart != nil {
		h.logger.Debugf(
			"body part: %s, %d",
			fmtError(m.HTTPResponseBodyPart.Error),
			len(m.HTTPResponseBodyPart.Data),
		)
	}
	if m.HTTPResponseDone != nil {
		h.logger.Debug(
			"end of response",
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
