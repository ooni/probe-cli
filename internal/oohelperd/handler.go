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
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/version"
	"golang.org/x/net/publicsuffix"
)

// maxAcceptableBodySize is the maximum acceptable body size for incoming
// API requests as well as when we're measuring webpages.
const maxAcceptableBodySize = 1 << 24

// Handler is an [http.Handler] implementing the Web
// Connectivity test helper HTTP API.
//
// The zero value is invalid; construct using [NewHandler].
type Handler struct {
	// EnableQUIC OPTIONALLY enables QUIC.
	EnableQUIC bool

	// baseLogger is the MANDATORY logger to use.
	baseLogger model.Logger

	// countRequests is the MANDATORY count of the number of
	// requests that are currently in flight.
	countRequests *atomic.Int64

	// indexer is the MANDATORY atomic integer used to assign an index to requests.
	indexer *atomic.Int64

	// maxAcceptableBody is the MANDATORY maximum acceptable response body.
	maxAcceptableBody int64

	// measure is the MANDATORY function that the handler should call
	// for producing a response for a valid incoming request.
	measure func(ctx context.Context, config *Handler, creq *model.THRequest) (*model.THResponse, error)

	// newDialer is the MANDATORY factory to create a new Dialer.
	newDialer func(model.Logger) model.Dialer

	// newHTTPClient is the MANDATORY factory to create a new HTTPClient.
	newHTTPClient func(model.Logger) model.HTTPClient

	// newHTTP3Client is the MANDATORY factory to create a new HTTP3Client.
	newHTTP3Client func(model.Logger) model.HTTPClient

	// newQUICDialer is the MANDATORY factory to create a new QUICDialer.
	newQUICDialer func(model.Logger) model.QUICDialer

	// newResolver is the MANDATORY factory for creating a new resolver.
	newResolver func(model.Logger) model.Resolver

	// newTLSHandshaker is the MANDATORY factory for creating a new TLS handshaker.
	newTLSHandshaker func(model.Logger) model.TLSHandshaker
}

var _ http.Handler = &Handler{}

// enableQUIC allows to control whether to enable QUIC by using environment variables.
var enableQUIC = (os.Getenv("OOHELPERD_ENABLE_QUIC") == "1")

// NewHandler constructs the [handler].
func NewHandler(logger model.Logger, netx *netxlite.Netx) *Handler {
	return &Handler{
		EnableQUIC:        enableQUIC,
		baseLogger:        logger,
		countRequests:     &atomic.Int64{},
		indexer:           &atomic.Int64{},
		maxAcceptableBody: maxAcceptableBodySize,
		measure:           measure,

		newHTTPClient: func(logger model.Logger) model.HTTPClient {
			// TODO(https://github.com/ooni/probe/issues/2534): the NewHTTPTransportWithResolver has QUIRKS and
			// we should evaluate whether we can avoid using it here
			return newHTTPClientWithTransportFactory(
				netx, logger,
				netxlite.NewHTTPTransportWithResolver,
			)
		},

		newHTTP3Client: func(logger model.Logger) model.HTTPClient {
			return newHTTPClientWithTransportFactory(
				netx, logger,
				netxlite.NewHTTP3TransportWithResolver,
			)
		},

		newDialer: func(logger model.Logger) model.Dialer {
			return netx.NewDialerWithoutResolver(logger)
		},

		newQUICDialer: func(logger model.Logger) model.QUICDialer {
			return netx.NewQUICDialerWithoutResolver(
				netx.NewUDPListener(),
				logger,
			)
		},

		newResolver: func(logger model.Logger) model.Resolver {
			return newResolver(logger, netx)
		},

		newTLSHandshaker: func(logger model.Logger) model.TLSHandshaker {
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

	// handle GET method for health check
	if req.Method == "GET" {
		metricRequestsCount.WithLabelValues("200", "ok").Inc()
		resp := map[string]string{
			"message": "Hello OONItarian!",
		}
		data, err := json.Marshal(resp)
		runtimex.PanicOnError(err, "json.Marshal failed")
		w.Header().Add("Content-Type", "application/json")
		w.Write(data)
		return
	}

	// we only handle the POST method for response generation
	if req.Method != "POST" {
		metricRequestsCount.WithLabelValues("400", "bad_request_method").Inc()
		w.WriteHeader(400)
		return
	}

	// protect against too many requests in flight
	if handlerShouldThrottleClient(h.countRequests.Load(), req.Header.Get("user-agent")) {
		metricRequestsCount.WithLabelValues("503", "service_unavailable").Inc()
		w.WriteHeader(503)
		return
	}
	h.countRequests.Add(1)
	defer h.countRequests.Add(-1)

	// read and parse request body
	reader := io.LimitReader(req.Body, h.maxAcceptableBody)
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
	cresp, err := h.measure(req.Context(), h, &creq)
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
func newResolver(logger model.Logger, netx *netxlite.Netx) model.Resolver {
	// Implementation note: pin to a specific resolver so we don't depend upon the
	// default resolver configured by the box. Also, use an encrypted transport thus
	// we're less vulnerable to any policy implemented by the box's provider.
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

// newHTTPClientWithTransportFactory creates a new HTTP client
// using the given [model.HTTPTransport] factory.
func newHTTPClientWithTransportFactory(
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
		newResolver(logger, netx),
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
