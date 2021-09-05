package netxlogger

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/apex/log"
	"github.com/apex/log/handlers/discard"
	"github.com/ooni/probe-cli/v3/internal/engine/legacy/netx"
	"github.com/ooni/probe-cli/v3/internal/engine/legacy/netx/modelx"
	"github.com/ooni/probe-cli/v3/internal/netxlite/iox"
)

func TestGood(t *testing.T) {
	log.SetHandler(discard.Default)
	client := netx.NewHTTPClient()
	client.ConfigureDNS("udp", "dns.google.com:53")
	req, err := http.NewRequest("GET", "http://www.facebook.com", nil)
	if err != nil {
		t.Fatal(err)
	}
	req = req.WithContext(modelx.WithMeasurementRoot(req.Context(), &modelx.MeasurementRoot{
		Beginning: time.Now(),
		Handler:   NewHandler(log.Log),
	}))
	resp, err := client.HTTPClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp == nil {
		t.Fatal("expected non-nil resp here")
	}
	defer resp.Body.Close()
	_, err = iox.ReadAllContext(context.Background(), resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	client.HTTPClient.CloseIdleConnections()
}
