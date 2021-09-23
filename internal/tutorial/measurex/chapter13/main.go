package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"time"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/measurex"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

type measurement struct {
	Queries       []*measurex.DNSLookupEvent     `json:"queries"`
	TCPConnect    []*measurex.NetworkEvent       `json:"tcp_connect"`
	TLSHandshakes []*measurex.TLSHandshakeEvent  `json:"tls_handshakes"`
	Requests      []*measurex.HTTPRoundTripEvent `json:"requests"`
}

func (m *measurement) addQueries(dm *measurex.DNSMeasurement) {
	m.Queries = append(m.Queries, dm.LookupHost...)
}

func (m *measurement) addEndpointCheck(em *measurex.EndpointMeasurement) {
	for _, ev := range em.Connect {
		switch ev.Network {
		case "tcp":
			m.TCPConnect = append(m.TCPConnect, ev)
		}
	}
	m.TLSHandshakes = append(m.TLSHandshakes, em.TLSHandshake...)
}

func (m *measurement) addHTTPCheck(hem *measurex.Measurement) {
	m.Requests = append(m.Requests, hem.HTTPRoundTrip...)
}

func main() {
	URL := flag.String("url", "https://www.google.com/", "URL to fetch")
	timeout := flag.Duration("timeout", 60*time.Second, "timeout to use")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	mx := measurex.NewMeasurerWithDefaultSettings()
	cookies := measurex.NewCookieJar()
	db := &measurex.MeasurementDB{}
	txp := mx.NewTracingHTTPTransportWithDefaultSettings(log.Log, db)
	txp.MaxBodySnapshotSize = 1 << 14
	client := &http.Client{Jar: cookies, Transport: txp}
	req, err := measurex.NewHTTPGetRequest(ctx, *URL)
	runtimex.PanicOnError(err, "NewHTTPGetRequest failed")
	resp, err := client.Do(req)
	if err == nil {
		resp.Body.Close() // be tidy
	}
	httpEndpoints, err := measurex.UnmeasuredHTTPEndpoints(
		db, *URL, measurex.NewHTTPRequestHeaderForMeasuring())
	runtimex.PanicOnError(err, "cannot determine unmeasured HTTP endpoints")
	for _, epnt := range httpEndpoints {
		resp, err = mx.HTTPEndpointGetWithDB(ctx, epnt, db, cookies)
		if err == nil {
			resp.Body.Close() // be tidy
		}
	}
	m := db.AsMeasurement()
	data, err := json.Marshal(m)
	runtimex.PanicOnError(err, "json.Marshal failed")
	fmt.Printf("%s\n", string(data))
}
