package oohelperd

//
// HTTP handler
//

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"sync/atomic"
	"time"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/version"
	"golang.org/x/net/publicsuffix"
)

// MaxAcceptableBodySize is the maximum acceptable body size for incoming
// API requests as well as when we're measuring webpages.
const MaxAcceptableBodySize = 1 << 24

// Handler is an [http.Handler] implementing the Web
// Connectivity test helper HTTP API.
type Handler struct {
	// BaseLogger is the MANDATORY logger to use.
	BaseLogger model.Logger

	// CountRequests is the MANDATORY count of the number of
	// requests that are currently in flight.
	CountRequests *atomic.Int64

	// Indexer is the MANDATORY atomic integer used to assign an index to requests.
	Indexer *atomic.Int64

	// MaxAcceptableBody is the MANDATORY maximum acceptable response body.
	MaxAcceptableBody int64

	// Measure is the MANDATORY function that the handler should call
	// for producing a response for a valid incoming request.
	Measure func(ctx context.Context, config *Handler, creq *model.THRequest) (*model.THResponse, error)

	// NewDialer is the MANDATORY factory to create a new Dialer.
	NewDialer func(model.Logger) model.Dialer

	// NewHTTPClient is the MANDATORY factory to create a new HTTPClient.
	NewHTTPClient func(model.Logger) model.HTTPClient

	// NewHTTP3Client is the MANDATORY factory to create a new HTTP3Client.
	NewHTTP3Client func(model.Logger) model.HTTPClient

	// NewQUICDialer is the MANDATORY factory to create a new QUICDialer.
	NewQUICDialer func(model.Logger) model.QUICDialer

	// NewResolver is the MANDATORY factory for creating a new resolver.
	NewResolver func(model.Logger) model.Resolver

	// NewTLSHandshaker is the MANDATORY factory for creating a new TLS handshaker.
	NewTLSHandshaker func(model.Logger) model.TLSHandshaker
}

var _ http.Handler = &Handler{}

// NewHandler constructs the [handler].
func NewHandler() *Handler {
	netx := &netxlite.Netx{}
	return &Handler{
		BaseLogger:        log.Log,
		CountRequests:     &atomic.Int64{},
		Indexer:           &atomic.Int64{},
		MaxAcceptableBody: MaxAcceptableBodySize,
		Measure:           measure,

		NewHTTPClient: func(logger model.Logger) model.HTTPClient {
			// TODO(https://github.com/ooni/probe/issues/2534): the NewHTTPTransportWithResolver has QUIRKS and
			// we should evaluate whether we can avoid using it here
			return NewHTTPClientWithTransportFactory(
				netx, logger,
				netxlite.NewHTTPTransportWithResolver,
			)
		},

		NewHTTP3Client: func(logger model.Logger) model.HTTPClient {
			return NewHTTPClientWithTransportFactory(
				netx, logger,
				netxlite.NewHTTP3TransportWithResolver,
			)
		},

		NewDialer: func(logger model.Logger) model.Dialer {
			return netx.NewDialerWithoutResolver(logger)
		},
		NewQUICDialer: func(logger model.Logger) model.QUICDialer {
			return netx.NewQUICDialerWithoutResolver(
				netx.NewUDPListener(),
				logger,
			)
		},
		NewResolver: newResolver,
		NewTLSHandshaker: func(logger model.Logger) model.TLSHandshaker {
			return netx.NewTLSHandshakerStdlib(logger)
		},
	}
}

// handlerShouldThrottleClient returns true if the handler should throttle
// the current client depending on the instantaneous load.
//
// See https://github.com/ooni/probe/issues/2649 for context.
func handlerShouldThrottleClient(inflight int64, userAgent string) bool {
	switch {
	// With less than 25 inflight requests we allow all clients
	case inflight < 25:
		return false

	// With less than 50 inflight requests we give priority to official clients
	case inflight < 50 && strings.HasPrefix(userAgent, "ooniprobe-"):
		return false

	// Otherwise, we're very sorry
	default:
		return true
	}
}

// ServeHTTP implements http.Handler.
func (h *Handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// track the number of in-flight requests
	metricRequestsInflight.Inc()
	defer metricRequestsInflight.Dec()

	// create and add the Server header
	w.Header().Add("Server", fmt.Sprintf(
		"oohelperd/%s ooniprobe-engine/%s",
		version.Version,
		version.Version,
	))

	// we only handle the POST method
	if req.Method != "POST" {
		metricRequestsCount.WithLabelValues("400", "bad_request_method").Inc()
		w.WriteHeader(400)
		return
	}

	// protect against too many requests in flight
	if handlerShouldThrottleClient(h.CountRequests.Load(), req.Header.Get("user-agent")) {
		metricRequestsCount.WithLabelValues("503", "service_unavailable").Inc()
		w.WriteHeader(503)
		return
	}
	h.CountRequests.Add(1)
	defer h.CountRequests.Add(-1)

	// read and parse request body
	reader := io.LimitReader(req.Body, h.MaxAcceptableBody)
	data, err := netxlite.ReadAllContext(req.Context(), reader)
	if err != nil {
		metricRequestsCount.WithLabelValues("400", "request_body_too_large").Inc()
		w.WriteHeader(400)
		return
	}
	var creq ctrlRequest
	if err := json.Unmarshal(data, &creq); err != nil {
		metricRequestsCount.WithLabelValues("400", "cannot_unmarshal_request_body").Inc()
		w.WriteHeader(400)
		return
	}

	// measure the given input
	started := time.Now()
	cresp, err := h.Measure(req.Context(), h, &creq)
	elapsed := time.Since(started)

	// track the time required to produce a response
	metricWCTaskDurationSeconds.Observe(elapsed.Seconds())

	// handle the case of fundamental failure
	if err != nil {
		metricRequestsCount.WithLabelValues("400", "wctask_failed").Inc()
		w.WriteHeader(400)
		return
	}

	// produce successful response.
	//
	// Note: we assume that json.Marshal cannot fail because it's a
	// clearly-serializable data structure.
	metricRequestsCount.WithLabelValues("200", "ok").Inc()
	data, err = json.Marshal(cresp)
	runtimex.PanicOnError(err, "json.Marshal failed")
	w.Header().Add("Content-Type", "application/json")
	w.Write(data)
}

// newResolver creates a new [model.Resolver] suitable for serving
// requests coming from ooniprobe clients.
func newResolver(logger model.Logger) model.Resolver {
	// Implementation note: pin to a specific resolver so we don't depend upon the
	// default resolver configured by the box. Also, use an encrypted transport thus
	// we're less vulnerable to any policy implemented by the box's provider.
	netx := &netxlite.Netx{}
	resolver := netx.NewParallelDNSOverHTTPSResolver(logger, "https://dns.google/dns-query")
	return resolver
}

// newCookieJar is the factory for constructing a new cookier jar.
func newCookieJar() *cookiejar.Jar {
	// Implementation note: the [cookiejar.New] function always returns a
	// nil error; hence, it's safe here to use [runtimex.Try1].
	return runtimex.Try1(cookiejar.New(&cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	}))
}

// NewHTTPClientWithTransportFactory creates a new HTTP client
// using the given [model.HTTPTransport] factory.
func NewHTTPClientWithTransportFactory(
	netx *netxlite.Netx, logger model.Logger,
	txpFactory func(*netxlite.Netx, model.DebugLogger, model.Resolver) model.HTTPTransport,
) model.HTTPClient {
	// If the DoH resolver we're using insists that a given domain maps to
	// bogons, make sure we're going to fail the HTTP measurement.
	//
	// The TCP measurements scheduler in ipinfo.go will also refuse to
	// schedule TCP measurements for bogons.
	//
	// While this seems theoretical, as of 2022-08-28, I see:
	//
	//     % host polito.it
	//     polito.it has address 192.168.59.6
	//     polito.it has address 192.168.40.1
	//     polito.it mail is handled by 10 mx.polito.it.
	//
	// So, it's better to consider this as a possible corner case.
	reso := netxlite.MaybeWrapWithBogonResolver(
		true, // enabled
		newResolver(logger),
	)

	// fix: We MUST set a cookie jar for measuring HTTP. See
	// https://github.com/ooni/probe/issues/2488 for additional
	// context and pointers to the relevant measurements.
	client := &http.Client{
		Transport:     txpFactory(netx, logger, reso),
		CheckRedirect: nil,
		Jar:           newCookieJar(),
		Timeout:       0,
	}

	return netxlite.WrapHTTPClient(client)
}
