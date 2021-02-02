package oldhttptransport

import (
	"bytes"
	"context"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptrace"
	"sync"
	"testing"
	"time"

	"github.com/miekg/dns"
	"github.com/ooni/probe-cli/v3/internal/engine/legacy/netx/modelx"
)

func TestTraceTripperSuccess(t *testing.T) {
	client := &http.Client{
		Transport: NewTraceTripper(http.DefaultTransport),
	}
	resp, err := client.Get("https://www.google.com")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	_, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	client.CloseIdleConnections()
}

type roundTripHandler struct {
	roundTrips []*modelx.HTTPRoundTripDoneEvent
	mu         sync.Mutex
}

func (h *roundTripHandler) OnMeasurement(m modelx.Measurement) {
	if m.HTTPRoundTripDone != nil {
		h.mu.Lock()
		defer h.mu.Unlock()
		h.roundTrips = append(h.roundTrips, m.HTTPRoundTripDone)
	}
}

func TestTraceTripperReadAllFailure(t *testing.T) {
	transport := NewTraceTripper(http.DefaultTransport)
	transport.readAll = func(r io.Reader) ([]byte, error) {
		return nil, io.EOF
	}
	client := &http.Client{Transport: transport}
	resp, err := client.Get("https://google.com")
	if err == nil {
		t.Fatal("expected an error here")
	}
	if !errors.Is(err, io.EOF) {
		t.Fatal("not the error we expected")
	}
	if resp != nil {
		t.Fatal("expected nil response here")
	}
	if transport.readAllErrs.Load() <= 0 {
		t.Fatal("not the error we expected")
	}
	client.CloseIdleConnections()
}

func TestTraceTripperFailure(t *testing.T) {
	client := &http.Client{
		Transport: NewTraceTripper(http.DefaultTransport),
	}
	// This fails the request because we attempt to speak cleartext HTTP with
	// a server that instead is expecting TLS.
	resp, err := client.Get("http://www.google.com:443")
	if err == nil {
		t.Fatal("expected an error here")
	}
	if resp != nil {
		t.Fatal("expected a nil response here")
	}
	client.CloseIdleConnections()
}

func TestTraceTripperWithClientTrace(t *testing.T) {
	client := &http.Client{
		Transport: NewTraceTripper(http.DefaultTransport),
	}
	req, err := http.NewRequest("GET", "https://www.kernel.org/", nil)
	if err != nil {
		t.Fatal(err)
	}
	req = req.WithContext(
		httptrace.WithClientTrace(req.Context(), new(httptrace.ClientTrace)),
	)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp == nil {
		t.Fatal("expected a good response here")
	}
	resp.Body.Close()
	client.CloseIdleConnections()
}

func TestTraceTripperWithCorrectSnaps(t *testing.T) {
	// Prepare a DNS query for dns.google.com A, for which we
	// know the answer in terms of well know IP addresses
	query := new(dns.Msg)
	query.Id = dns.Id()
	query.RecursionDesired = true
	query.Question = make([]dns.Question, 1)
	query.Question[0] = dns.Question{
		Name:   dns.Fqdn("dns.google.com"),
		Qtype:  dns.TypeA,
		Qclass: dns.ClassINET,
	}
	queryData, err := query.Pack()
	if err != nil {
		t.Fatal(err)
	}

	// Prepare a new transport with limited snapshot size and
	// use such transport to configure an ordinary client
	transport := NewTraceTripper(http.DefaultTransport)
	const snapSize = 15
	client := &http.Client{Transport: transport}

	// Prepare a new request for Cloudflare DNS, register
	// a handler, issue the request, fetch the response.
	req, err := http.NewRequest(
		"POST", "https://cloudflare-dns.com/dns-query", bytes.NewReader(queryData),
	)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/dns-message")
	handler := &roundTripHandler{}
	ctx := modelx.WithMeasurementRoot(
		context.Background(), &modelx.MeasurementRoot{
			Beginning:       time.Now(),
			Handler:         handler,
			MaxBodySnapSize: snapSize,
		},
	)
	req = req.WithContext(ctx)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Fatal("HTTP request failed")
	}

	// Read the whole response body, parse it as valid DNS
	// reply and verify we obtained what we expected
	replyData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	reply := new(dns.Msg)
	err = reply.Unpack(replyData)
	if err != nil {
		t.Fatal(err)
	}
	if reply.Rcode != 0 {
		t.Fatal("unexpected Rcode")
	}
	if len(reply.Answer) < 1 {
		t.Fatal("no answers?!")
	}
	found8888, found8844, foundother := false, false, false
	for _, answer := range reply.Answer {
		if rra, ok := answer.(*dns.A); ok {
			ip := rra.A.String()
			if ip == "8.8.8.8" {
				found8888 = true
			} else if ip == "8.8.4.4" {
				found8844 = true
			} else {
				foundother = true
			}
		}
	}
	if !found8888 || !found8844 || foundother {
		t.Fatal("unexpected reply")
	}

	// Finally, make sure we have captured the correct
	// snapshots for the request and response bodies
	if len(handler.roundTrips) != 1 {
		t.Fatal("more round trips than expected")
	}
	roundTrip := handler.roundTrips[0]
	if len(roundTrip.RequestBodySnap) != snapSize {
		t.Fatal("unexpected request body snap length")
	}
	if len(roundTrip.ResponseBodySnap) != snapSize {
		t.Fatal("unexpected response body snap length")
	}
	if !bytes.Equal(roundTrip.RequestBodySnap, queryData[:snapSize]) {
		t.Fatal("the request body snap is wrong")
	}
	if !bytes.Equal(roundTrip.ResponseBodySnap, replyData[:snapSize]) {
		t.Fatal("the response body snap is wrong")
	}
}

func TestTraceTripperWithReadAllFailingForBody(t *testing.T) {
	// Prepare a DNS query for dns.google.com A, for which we
	// know the answer in terms of well know IP addresses
	query := new(dns.Msg)
	query.Id = dns.Id()
	query.RecursionDesired = true
	query.Question = make([]dns.Question, 1)
	query.Question[0] = dns.Question{
		Name:   dns.Fqdn("dns.google.com"),
		Qtype:  dns.TypeA,
		Qclass: dns.ClassINET,
	}
	queryData, err := query.Pack()
	if err != nil {
		t.Fatal(err)
	}

	// Prepare a new transport with limited snapshot size and
	// use such transport to configure an ordinary client
	transport := NewTraceTripper(http.DefaultTransport)
	errorMocked := errors.New("mocked error")
	transport.readAll = func(r io.Reader) ([]byte, error) {
		return nil, errorMocked
	}
	const snapSize = 15
	client := &http.Client{Transport: transport}

	// Prepare a new request for Cloudflare DNS, register
	// a handler, issue the request, fetch the response.
	req, err := http.NewRequest(
		"POST", "https://cloudflare-dns.com/dns-query", bytes.NewReader(queryData),
	)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/dns-message")
	handler := &roundTripHandler{}
	ctx := modelx.WithMeasurementRoot(
		context.Background(), &modelx.MeasurementRoot{
			Beginning:       time.Now(),
			Handler:         handler,
			MaxBodySnapSize: snapSize,
		},
	)
	req = req.WithContext(ctx)
	resp, err := client.Do(req)
	if err == nil {
		t.Fatal("expected an error here")
	}
	if !errors.Is(err, errorMocked) {
		t.Fatal("not the error we expected")
	}
	if resp != nil {
		t.Fatal("expected nil response here")
	}

	// Finally, make sure we got something that makes sense
	if len(handler.roundTrips) != 0 {
		t.Fatal("more round trips than expected")
	}
}
