package urlgetter_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/urlgetter"
	"github.com/ooni/probe-cli/v3/internal/engine/mockable"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func TestGetterWithVeryShortTimeout(t *testing.T) {
	g := urlgetter.Getter{
		Config: urlgetter.Config{
			Timeout: 1,
		},
		Session: &mockable.Session{},
		Target:  "https://www.google.com",
	}
	tk, err := g.Get(context.Background())
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal("not the error we expected")
	}
	if tk.Agent != "redirect" {
		t.Fatal("not the Agent we expected")
	}
	if tk.BootstrapTime != 0 {
		t.Fatal("not the BootstrapTime we expected")
	}
	if tk.FailedOperation == nil || *tk.FailedOperation != netxlite.TopLevelOperation {
		t.Fatal("not the FailedOperation we expected")
	}
	if tk.Failure == nil || *tk.Failure != "generic_timeout_error" {
		t.Fatal("not the Failure we expected")
	}
	if len(tk.NetworkEvents) != 2 {
		t.Fatal("not the NetworkEvents we expected")
	}
	if tk.NetworkEvents[0].Operation != "http_transaction_start" {
		t.Fatal("not the NetworkEvents[0].Operation we expected")
	}
	if tk.NetworkEvents[1].Operation != "http_transaction_done" {
		t.Fatal("not the NetworkEvents[1].Operation we expected")
	}
	if len(tk.Queries) != 0 {
		t.Fatal("not the Queries we expected")
	}
	if len(tk.TCPConnect) != 0 {
		t.Fatal("not the TCPConnect we expected")
	}
	if len(tk.Requests) != 1 {
		t.Fatal("not the Requests we expected")
	}
	if tk.Requests[0].Request.Method != "GET" {
		t.Fatal("not the Method we expected")
	}
	if tk.Requests[0].Request.URL != "https://www.google.com" {
		t.Fatal("not the URL we expected")
	}
	if tk.SOCKSProxy != "" {
		t.Fatal("not the SOCKSProxy we expected")
	}
	if len(tk.TLSHandshakes) != 0 {
		t.Fatal("not the TLSHandshakes we expected")
	}
	if tk.Tunnel != "" {
		t.Fatal("not the Tunnel we expected")
	}
	if tk.HTTPResponseStatus != 0 {
		t.Fatal("not the HTTPResponseStatus we expected")
	}
	if tk.HTTPResponseBody != "" {
		t.Fatal("not the HTTPResponseBody we expected")
	}
}

func TestGetterWithCancelledContextVanilla(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // faily immediately
	g := urlgetter.Getter{
		Session: &mockable.Session{},
		Target:  "https://www.google.com",
	}
	tk, err := g.Get(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatal("not the error we expected")
	}
	if tk.Agent != "redirect" {
		t.Fatal("not the Agent we expected")
	}
	if tk.BootstrapTime != 0 {
		t.Fatal("not the BootstrapTime we expected")
	}
	if tk.FailedOperation == nil || *tk.FailedOperation != netxlite.TopLevelOperation {
		t.Fatal("not the FailedOperation we expected")
	}
	if tk.Failure == nil || !strings.HasSuffix(*tk.Failure, "interrupted") {
		t.Fatal("not the Failure we expected")
	}
	if len(tk.NetworkEvents) != 2 {
		t.Fatal("not the NetworkEvents we expected")
	}
	if tk.NetworkEvents[0].Operation != "http_transaction_start" {
		t.Fatal("not the NetworkEvents[0].Operation we expected")
	}
	if tk.NetworkEvents[1].Operation != "http_transaction_done" {
		t.Fatal("not the NetworkEvents[1].Operation we expected")
	}
	if len(tk.Queries) != 0 {
		t.Fatal("not the Queries we expected")
	}
	if len(tk.TCPConnect) != 0 {
		t.Fatal("not the TCPConnect we expected")
	}
	if len(tk.Requests) != 1 {
		t.Fatal("not the Requests we expected")
	}
	if tk.Requests[0].Request.Method != "GET" {
		t.Fatal("not the Method we expected")
	}
	if tk.Requests[0].Request.URL != "https://www.google.com" {
		t.Fatal("not the URL we expected")
	}
	if tk.SOCKSProxy != "" {
		t.Fatal("not the SOCKSProxy we expected")
	}
	if len(tk.TLSHandshakes) != 0 {
		t.Fatal("not the TLSHandshakes we expected")
	}
	if tk.Tunnel != "" {
		t.Fatal("not the Tunnel we expected")
	}
	if tk.HTTPResponseStatus != 0 {
		t.Fatal("not the HTTPResponseStatus we expected")
	}
	if tk.HTTPResponseBody != "" {
		t.Fatal("not the HTTPResponseBody we expected")
	}
}

func TestGetterWithCancelledContextAndMethod(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // faily immediately
	g := urlgetter.Getter{
		Config:  urlgetter.Config{Method: "POST"},
		Session: &mockable.Session{},
		Target:  "https://www.google.com",
	}
	tk, err := g.Get(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatal("not the error we expected")
	}
	if tk.Agent != "redirect" {
		t.Fatal("not the Agent we expected")
	}
	if tk.BootstrapTime != 0 {
		t.Fatal("not the BootstrapTime we expected")
	}
	if tk.FailedOperation == nil || *tk.FailedOperation != netxlite.TopLevelOperation {
		t.Fatal("not the FailedOperation we expected")
	}
	if tk.Failure == nil || !strings.HasSuffix(*tk.Failure, "interrupted") {
		t.Fatal("not the Failure we expected")
	}
	if len(tk.NetworkEvents) != 2 {
		t.Fatal("not the NetworkEvents we expected")
	}
	if tk.NetworkEvents[0].Operation != "http_transaction_start" {
		t.Fatal("not the NetworkEvents[0].Operation we expected")
	}
	if tk.NetworkEvents[1].Operation != "http_transaction_done" {
		t.Fatal("not the NetworkEvents[1].Operation we expected")
	}
	if len(tk.Queries) != 0 {
		t.Fatal("not the Queries we expected")
	}
	if len(tk.TCPConnect) != 0 {
		t.Fatal("not the TCPConnect we expected")
	}
	if len(tk.Requests) != 1 {
		t.Fatal("not the Requests we expected")
	}
	if tk.Requests[0].Request.Method != "POST" {
		t.Fatal("not the Method we expected")
	}
	if tk.Requests[0].Request.URL != "https://www.google.com" {
		t.Fatal("not the URL we expected")
	}
	if tk.SOCKSProxy != "" {
		t.Fatal("not the SOCKSProxy we expected")
	}
	if len(tk.TLSHandshakes) != 0 {
		t.Fatal("not the TLSHandshakes we expected")
	}
	if tk.Tunnel != "" {
		t.Fatal("not the Tunnel we expected")
	}
	if tk.HTTPResponseStatus != 0 {
		t.Fatal("not the HTTPResponseStatus we expected")
	}
	if tk.HTTPResponseBody != "" {
		t.Fatal("not the HTTPResponseBody we expected")
	}
}

func TestGetterWithCancelledContextNoFollowRedirects(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // faily immediately
	g := urlgetter.Getter{
		Config: urlgetter.Config{
			NoFollowRedirects: true,
		},
		Session: &mockable.Session{},
		Target:  "https://www.google.com",
	}
	tk, err := g.Get(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatal("not the error we expected")
	}
	if tk.Agent != "agent" {
		t.Fatal("not the Agent we expected")
	}
	if tk.BootstrapTime != 0 {
		t.Fatal("not the BootstrapTime we expected")
	}
	if tk.FailedOperation == nil || *tk.FailedOperation != netxlite.TopLevelOperation {
		t.Fatal("not the FailedOperation we expected")
	}
	if tk.Failure == nil || !strings.HasSuffix(*tk.Failure, "interrupted") {
		t.Fatal("not the Failure we expected")
	}
	if len(tk.NetworkEvents) != 2 {
		t.Fatal("not the NetworkEvents we expected")
	}
	if tk.NetworkEvents[0].Operation != "http_transaction_start" {
		t.Fatal("not the NetworkEvents[0].Operation we expected")
	}
	if tk.NetworkEvents[1].Operation != "http_transaction_done" {
		t.Fatal("not the NetworkEvents[2].Operation we expected")
	}
	if len(tk.Queries) != 0 {
		t.Fatal("not the Queries we expected")
	}
	if len(tk.TCPConnect) != 0 {
		t.Fatal("not the TCPConnect we expected")
	}
	if len(tk.Requests) != 1 {
		t.Fatal("not the Requests we expected")
	}
	if tk.Requests[0].Request.Method != "GET" {
		t.Fatal("not the Method we expected")
	}
	if tk.Requests[0].Request.URL != "https://www.google.com" {
		t.Fatal("not the URL we expected")
	}
	if tk.SOCKSProxy != "" {
		t.Fatal("not the SOCKSProxy we expected")
	}
	if len(tk.TLSHandshakes) != 0 {
		t.Fatal("not the TLSHandshakes we expected")
	}
	if tk.Tunnel != "" {
		t.Fatal("not the Tunnel we expected")
	}
	if tk.HTTPResponseStatus != 0 {
		t.Fatal("not the HTTPResponseStatus we expected")
	}
	if tk.HTTPResponseBody != "" {
		t.Fatal("not the HTTPResponseBody we expected")
	}
}

func TestGetterWithCancelledContextCannotStartTunnel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // fail immediately
	g := urlgetter.Getter{
		Config: urlgetter.Config{
			Tunnel: "psiphon",
		},
		Session: &mockable.Session{MockableLogger: log.Log},
		Target:  "https://www.google.com",
	}
	tk, err := g.Get(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("not the error we expected: %+v", err)
	}
	if tk.Agent != "redirect" {
		t.Fatal("not the Agent we expected")
	}
	if tk.BootstrapTime != 0 {
		t.Fatal("not the BootstrapTime we expected")
	}
	if tk.FailedOperation == nil || *tk.FailedOperation != netxlite.TopLevelOperation {
		t.Fatal("not the FailedOperation we expected")
	}
	if tk.Failure == nil || *tk.Failure != "interrupted" {
		t.Fatal("not the Failure we expected")
	}
	if len(tk.NetworkEvents) != 0 {
		t.Fatal("not the NetworkEvents we expected")
	}
	if len(tk.Queries) != 0 {
		t.Fatal("not the Queries we expected")
	}
	if len(tk.TCPConnect) != 0 {
		t.Fatal("not the TCPConnect we expected")
	}
	if len(tk.Requests) != 0 {
		t.Fatal("not the Requests we expected")
	}
	if tk.SOCKSProxy != "" {
		t.Fatal("not the SOCKSProxy we expected")
	}
	if len(tk.TLSHandshakes) != 0 {
		t.Fatal("not the TLSHandshakes we expected")
	}
	if tk.Tunnel != "psiphon" {
		t.Fatal("not the Tunnel we expected")
	}
	if tk.HTTPResponseStatus != 0 {
		t.Fatal("not the HTTPResponseStatus we expected")
	}
	if tk.HTTPResponseBody != "" {
		t.Fatal("not the HTTPResponseBody we expected")
	}
}

func TestGetterWithCancelledContextUnknownResolverURL(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // faily immediately
	g := urlgetter.Getter{
		Config: urlgetter.Config{
			ResolverURL: "antani://8.8.8.8:53",
		},
		Session: &mockable.Session{},
		Target:  "https://www.google.com",
	}
	tk, err := g.Get(ctx)
	if err == nil || err.Error() != "unknown_failure: unsupported resolver scheme" {
		t.Fatal("not the error we expected")
	}
	if tk.Agent != "redirect" {
		t.Fatal("not the Agent we expected")
	}
	if tk.BootstrapTime != 0 {
		t.Fatal("not the BootstrapTime we expected")
	}
	if tk.FailedOperation == nil || *tk.FailedOperation != netxlite.TopLevelOperation {
		t.Fatal("not the FailedOperation we expected")
	}
	if tk.Failure == nil || *tk.Failure != "unknown_failure: unsupported resolver scheme" {
		t.Fatal("not the Failure we expected")
	}
	if len(tk.NetworkEvents) != 0 {
		t.Fatal("not the NetworkEvents we expected")
	}
	if len(tk.Queries) != 0 {
		t.Fatal("not the Queries we expected")
	}
	if len(tk.TCPConnect) != 0 {
		t.Fatal("not the TCPConnect we expected")
	}
	if len(tk.Requests) != 0 {
		t.Fatal("not the Requests we expected")
	}
	if tk.SOCKSProxy != "" {
		t.Fatal("not the SOCKSProxy we expected")
	}
	if len(tk.TLSHandshakes) != 0 {
		t.Fatal("not the TLSHandshakes we expected")
	}
	if tk.Tunnel != "" {
		t.Fatal("not the Tunnel we expected")
	}
	if tk.HTTPResponseStatus != 0 {
		t.Fatal("not the HTTPResponseStatus we expected")
	}
	if tk.HTTPResponseBody != "" {
		t.Fatal("not the HTTPResponseBody we expected")
	}
}

func TestGetterIntegrationHTTPS(t *testing.T) {
	ctx := context.Background()
	g := urlgetter.Getter{
		Config: urlgetter.Config{
			NoFollowRedirects: true, // reduce number of events
		},
		Session: &mockable.Session{},
		Target:  "https://www.google.com",
	}
	tk, err := g.Get(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if tk.Agent != "agent" {
		t.Fatal("not the Agent we expected")
	}
	if tk.BootstrapTime != 0 {
		t.Fatal("not the BootstrapTime we expected")
	}
	if tk.FailedOperation != nil {
		t.Fatal("not the FailedOperation we expected")
	}
	if tk.Failure != nil {
		t.Fatal("not the Failure we expected")
	}
	var (
		httpTransactionStart bool
		resolveStart         bool
		resolveDone          bool
		connect              bool
		tlsHandshakeStart    bool
		tlsHandshakeDone     bool
		httpTransactionDone  bool
	)
	for _, ev := range tk.NetworkEvents {
		switch ev.Operation {
		case "http_transaction_start":
			httpTransactionStart = true
		case "resolve_start":
			resolveStart = true
		case "resolve_done":
			resolveDone = true
		case netxlite.ConnectOperation:
			connect = true
		case "tls_handshake_start":
			tlsHandshakeStart = true
		case "tls_handshake_done":
			tlsHandshakeDone = true
		case "http_transaction_done":
			httpTransactionDone = true
		}
	}
	ok := true
	ok = ok && httpTransactionStart
	ok = ok && resolveStart
	ok = ok && resolveDone
	ok = ok && connect
	ok = ok && tlsHandshakeStart
	ok = ok && tlsHandshakeDone
	ok = ok && httpTransactionDone
	if !ok {
		t.Fatal("not the NetworkEvents we expected")
	}
	if len(tk.Queries) != 2 {
		t.Fatal("not the Queries we expected")
	}
	if len(tk.TCPConnect) != 1 {
		t.Fatal("not the TCPConnect we expected")
	}
	if len(tk.Requests) != 1 {
		t.Fatal("not the Requests we expected")
	}
	if tk.Requests[0].Request.Method != "GET" {
		t.Fatal("not the Method we expected")
	}
	if tk.Requests[0].Request.URL != "https://www.google.com" {
		t.Fatal("not the URL we expected")
	}
	if tk.SOCKSProxy != "" {
		t.Fatal("not the SOCKSProxy we expected")
	}
	if len(tk.TLSHandshakes) != 1 {
		t.Fatal("not the TLSHandshakes we expected")
	}
	if tk.Tunnel != "" {
		t.Fatal("not the Tunnel we expected")
	}
	if tk.HTTPResponseStatus != 200 {
		t.Fatal("not the HTTPResponseStatus we expected")
	}
	if len(tk.HTTPResponseBody) <= 0 {
		t.Fatal("not the HTTPResponseBody we expected")
	}
}

func TestGetterIntegrationRedirect(t *testing.T) {
	// Because of https://github.com/ooni/probe/issues/2374, we're now
	// using a local testing server (more robust anyway).
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Location", "https://web.whatsapp.com/")
		w.WriteHeader(302)
	}))
	defer server.Close()
	ctx := context.Background()
	g := urlgetter.Getter{
		Config:  urlgetter.Config{NoFollowRedirects: true},
		Session: &mockable.Session{},
		Target:  server.URL,
	}
	tk, err := g.Get(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if tk.HTTPResponseStatus != 302 {
		t.Fatal("unexpected status code")
	}
	if len(tk.HTTPResponseLocations) != 1 {
		t.Fatal("missing redirect URL")
	}
	if tk.HTTPResponseLocations[0] != "https://web.whatsapp.com/" {
		t.Fatal("invalid redirect URL")
	}
}

func TestGetterIntegrationTLSHandshake(t *testing.T) {
	ctx := context.Background()
	g := urlgetter.Getter{
		Config: urlgetter.Config{
			NoFollowRedirects: true, // reduce number of events
		},
		Session: &mockable.Session{},
		Target:  "tlshandshake://www.google.com:443",
	}
	tk, err := g.Get(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if tk.Agent != "agent" {
		t.Fatal("not the Agent we expected")
	}
	if tk.BootstrapTime != 0 {
		t.Fatal("not the BootstrapTime we expected")
	}
	if tk.FailedOperation != nil {
		t.Fatal("not the FailedOperation we expected")
	}
	if tk.Failure != nil {
		t.Fatal("not the Failure we expected")
	}
	var (
		httpTransactionStart     bool
		httpRequestMetadata      bool
		resolveStart             bool
		resolveDone              bool
		connect                  bool
		tlsHandshakeStart        bool
		tlsHandshakeDone         bool
		httpResponseMetadata     bool
		httpResponseBodySnapshot bool
		httpTransactionDone      bool
	)
	for _, ev := range tk.NetworkEvents {
		switch ev.Operation {
		case "http_transaction_start":
			httpTransactionStart = true
		case "http_request_metadata":
			httpRequestMetadata = true
		case "resolve_start":
			resolveStart = true
		case "resolve_done":
			resolveDone = true
		case netxlite.ConnectOperation:
			connect = true
		case "tls_handshake_start":
			tlsHandshakeStart = true
		case "tls_handshake_done":
			tlsHandshakeDone = true
		case "http_response_metadata":
			httpResponseMetadata = true
		case "http_response_body_snapshot":
			httpResponseBodySnapshot = true
		case "http_transaction_done":
			httpTransactionDone = true
		}
	}
	ok := true
	ok = ok && !httpTransactionStart
	ok = ok && !httpRequestMetadata
	ok = ok && resolveStart
	ok = ok && resolveDone
	ok = ok && connect
	ok = ok && tlsHandshakeStart
	ok = ok && tlsHandshakeDone
	ok = ok && !httpResponseMetadata
	ok = ok && !httpResponseBodySnapshot
	ok = ok && !httpTransactionDone
	if !ok {
		t.Fatal("not the NetworkEvents we expected")
	}
	if len(tk.Queries) != 2 {
		t.Fatal("not the Queries we expected")
	}
	if len(tk.TCPConnect) != 1 {
		t.Fatal("not the TCPConnect we expected")
	}
	if len(tk.Requests) != 0 {
		t.Fatal("not the Requests we expected")
	}
	if tk.SOCKSProxy != "" {
		t.Fatal("not the SOCKSProxy we expected")
	}
	if len(tk.TLSHandshakes) != 1 {
		t.Fatal("not the TLSHandshakes we expected")
	}
	if tk.Tunnel != "" {
		t.Fatal("not the Tunnel we expected")
	}
	if tk.HTTPResponseStatus != 0 {
		t.Fatal("not the HTTPResponseStatus we expected")
	}
	if tk.HTTPResponseBody != "" {
		t.Fatal("not the HTTPResponseBody we expected")
	}
}

func TestGetterHTTPSWithTunnel(t *testing.T) {
	// quick enough (0.4s) to run with every run
	ctx := context.Background()
	g := urlgetter.Getter{
		Config: urlgetter.Config{
			NoFollowRedirects: true, // reduce number of events
			Tunnel:            "fake",
		},
		Session: &mockable.Session{
			MockableHTTPClient: http.DefaultClient,
			MockableLogger:     log.Log,
		},
		Target: "https://www.google.com",
	}
	tk, err := g.Get(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if tk.Agent != "agent" {
		t.Fatal("not the Agent we expected")
	}
	if tk.BootstrapTime <= 0 {
		t.Fatal("not the BootstrapTime we expected")
	}
	if tk.FailedOperation != nil {
		t.Fatal("not the FailedOperation we expected")
	}
	if tk.Failure != nil {
		t.Fatal("not the Failure we expected")
	}
	var (
		httpTransactionStart bool
		resolveStart         bool
		resolveDone          bool
		connect              bool
		tlsHandshakeStart    bool
		tlsHandshakeDone     bool
		httpTransactionDone  bool
	)
	for _, ev := range tk.NetworkEvents {
		switch ev.Operation {
		case "http_transaction_start":
			httpTransactionStart = true
		case "resolve_start":
			resolveStart = true
		case "resolve_done":
			resolveDone = true
		case netxlite.ConnectOperation:
			connect = true
		case "tls_handshake_start":
			tlsHandshakeStart = true
		case "tls_handshake_done":
			tlsHandshakeDone = true
		case "http_transaction_done":
			httpTransactionDone = true
		}
	}
	ok := true
	ok = ok && httpTransactionStart
	ok = ok && resolveStart == false
	ok = ok && resolveDone == false
	ok = ok && connect
	ok = ok && tlsHandshakeStart
	ok = ok && tlsHandshakeDone
	ok = ok && httpTransactionDone
	if !ok {
		t.Fatalf("not the NetworkEvents we expected: %+v", tk.NetworkEvents)
	}
	if len(tk.Queries) != 0 {
		t.Fatal("not the Queries we expected")
	}
	if len(tk.TCPConnect) != 1 {
		t.Fatal("not the TCPConnect we expected")
	}
	if len(tk.Requests) != 1 {
		t.Fatal("not the Requests we expected")
	}
	if tk.Requests[0].Request.Method != "GET" {
		t.Fatal("not the Method we expected")
	}
	if tk.Requests[0].Request.URL != "https://www.google.com" {
		t.Fatal("not the URL we expected")
	}
	if tk.SOCKSProxy == "" {
		t.Fatal("not the SOCKSProxy we expected")
	}
	if len(tk.TLSHandshakes) != 1 {
		t.Fatal("not the TLSHandshakes we expected")
	}
	if tk.Tunnel != "fake" {
		t.Fatal("not the Tunnel we expected")
	}
	if tk.HTTPResponseStatus != 200 {
		t.Fatal("not the HTTPResponseStatus we expected")
	}
	if len(tk.HTTPResponseBody) <= 0 {
		t.Fatal("not the HTTPResponseBody we expected")
	}
}
