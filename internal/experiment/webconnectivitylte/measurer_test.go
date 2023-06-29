package webconnectivitylte_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/apex/log"
	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/engine"
	"github.com/ooni/probe-cli/v3/internal/experiment/webconnectivitylte"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netemx"
	"github.com/ooni/probe-cli/v3/internal/oohelperd"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

func TestSecureFlow_Run(t *testing.T) {
	type fields struct {
		Address         string
		DNSCache        *webconnectivitylte.DNSCache
		IDGenerator     *atomic.Int64
		Logger          model.Logger
		NumRedirects    *webconnectivitylte.NumRedirects
		TestKeys        *webconnectivitylte.TestKeys
		ZeroTime        time.Time
		WaitGroup       *sync.WaitGroup
		ALPN            []string
		CookieJar       http.CookieJar
		FollowRedirects bool
		HostHeader      string
		PrioSelector    *webconnectivitylte.PrioritySelector
		Referer         string
		SNI             string
		UDPAddress      string
		URLPath         string
		URLRawQuery     string
	}
	type args struct {
		parentCtx context.Context
		index     int64
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   error
	}{{
		name: "with loopback IPv4 endpoint",
		fields: fields{
			Address: "127.0.0.1:443",
			Logger:  model.DiscardLogger,
		},
		args: args{
			parentCtx: context.Background(),
			index:     0,
		},
		want: webconnectivitylte.ErrNotAllowedToConnect,
	}, {
		name: "with loopback IPv6 endpoint",
		fields: fields{
			Address: "[::1]:443",
			Logger:  model.DiscardLogger,
		},
		args: args{
			parentCtx: context.Background(),
			index:     0,
		},
		want: webconnectivitylte.ErrNotAllowedToConnect,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &webconnectivitylte.SecureFlow{
				Address:         tt.fields.Address,
				DNSCache:        tt.fields.DNSCache,
				IDGenerator:     tt.fields.IDGenerator,
				Logger:          tt.fields.Logger,
				NumRedirects:    tt.fields.NumRedirects,
				TestKeys:        tt.fields.TestKeys,
				ZeroTime:        tt.fields.ZeroTime,
				WaitGroup:       tt.fields.WaitGroup,
				ALPN:            tt.fields.ALPN,
				CookieJar:       tt.fields.CookieJar,
				FollowRedirects: tt.fields.FollowRedirects,
				HostHeader:      tt.fields.HostHeader,
				PrioSelector:    tt.fields.PrioSelector,
				Referer:         tt.fields.Referer,
				SNI:             tt.fields.SNI,
				UDPAddress:      tt.fields.UDPAddress,
				URLPath:         tt.fields.URLPath,
				URLRawQuery:     tt.fields.URLRawQuery,
			}
			err := tr.Run(tt.args.parentCtx, tt.args.index)
			if !errors.Is(err, tt.want) {
				t.Errorf("SecureFlow.Run() error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestSuccess(t *testing.T) {
	// configure default UDP DNS server
	dnsConfig := netem.NewDNSConfig()
	dnsConfig.AddRecord(
		"dns.quad9.net",
		"dns.quad9.net",
		"104.16.248.249",
	)
	dnsConfig.AddRecord(
		"mozilla.cloudflare-dns.com",
		"mozilla.cloudflare-dns.com",
		"104.16.248.249",
	)
	dnsConfig.AddRecord(
		"dns.nextdns.io",
		"dns.nextdns.io",
		"104.16.248.249",
	)
	dnsConfig.AddRecord(
		"dns.google",
		"dns.google",
		"104.16.248.249",
	)
	dnsConfig.AddRecord(
		"www.example.com",
		"www.example.com",
		"93.184.216.34",
	)

	// configure DoH server
	dohServer := &netemx.DoHServer{}
	dohServer.AddRecord("ams-pg-test.ooni.org", "188.166.93.143")
	dohServer.AddRecord("geoip.ubuntu.com", "185.125.188.132")
	dohServer.AddRecord("www.example.com", "93.184.216.34")
	dohServer.AddRecord("0.th.ooni.org", "104.248.30.161")
	dohServer.AddRecord("1.th.ooni.org", "104.248.30.161")
	dohServer.AddRecord("2.th.ooni.org", "104.248.30.161")
	dohServer.AddRecord("3.th.ooni.org", "104.248.30.161")

	// client config with DNS server at 8.8.4.4
	clientConf := &netemx.ClientConfig{
		DNSConfig:    dnsConfig,
		ResolverAddr: "8.8.4.4",
	}

	// servers config contains
	serversConf := &netemx.ServersConfig{
		DNSConfig: dnsConfig,
		Servers: []netemx.ConfigServerStack{
			{
				ServerAddr:  "13.13.13.13",
				HTTPServers: []netemx.ConfigHTTPServer{{Port: 443}},
			},
			{
				ServerAddr: "104.248.30.161",
				HTTPServers: []netemx.ConfigHTTPServer{
					{
						Port:    443,
						Handler: oohelperd.NewHandler(),
					},
				},
			},
			{
				ServerAddr: "104.16.248.249",
				HTTPServers: []netemx.ConfigHTTPServer{
					{
						Port:    443,
						Handler: dohServer,
					},
				},
			},
			{
				ServerAddr: "188.166.93.143",
				HTTPServers: []netemx.ConfigHTTPServer{
					{
						Port:    443,
						Handler: &probeService{},
					},
				},
			},
			{
				ServerAddr: "185.125.188.132",
				HTTPServers: []netemx.ConfigHTTPServer{
					{
						Port:    443,
						Handler: &netemx.GeoIPLookup{},
					},
				},
			},
		},
	}
	// create a new test environment
	env := netemx.NewEnvironment(clientConf, serversConf)
	defer env.Close()
	env.Do(func() {
		measurer := webconnectivitylte.NewExperimentMeasurer(&webconnectivitylte.Config{})
		ctx := context.Background()
		// we need a real session because we need the web-connectivity helper
		// as well as the ASN database
		sess := newsession(t, true)
		measurement := &model.Measurement{Input: "https://www.example.com"}
		callbacks := model.NewPrinterCallbacks(log.Log)
		args := &model.ExperimentArgs{
			Callbacks:   callbacks,
			Measurement: measurement,
			Session:     sess,
		}
		err := measurer.Run(ctx, args)
		if err != nil {
			t.Fatal(err)
		}
		tk := measurement.TestKeys.(*webconnectivitylte.TestKeys)
		if tk.ControlFailure != nil {
			t.Fatal("unexpected control_failure", *tk.ControlFailure)
		}
		if tk.DNSExperimentFailure != nil {
			t.Fatal("unexpected dns_experiment_failure", *tk.DNSExperimentFailure)
		}
		if tk.HTTPExperimentFailure != nil {
			t.Fatal("unexpected http_experiment_failure", *tk.HTTPExperimentFailure)
		}
	})
}

type probeService struct{}

type th struct {
	Addr string `json:"address"`
	T    string `json:"type"`
}

func (p *probeService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	resp := map[string][]th{
		"web-connectivity": {
			{
				Addr: "https://2.th.ooni.org",
				T:    "https",
			},
			{
				Addr: "https://3.th.ooni.org",
				T:    "https",
			},
			{
				Addr: "https://0.th.ooni.org",
				T:    "https",
			},
			{
				Addr: "https://1.th.ooni.org",
				T:    "https",
			},
		},
	}
	data, err := json.Marshal(resp)
	runtimex.PanicOnError(err, "json.Marshal failed")
	w.Header().Add("Content-Type", "application/json")
	w.Write(data)
}

func newsession(t *testing.T, lookupBackends bool) model.ExperimentSession {
	sess, err := engine.NewSession(context.Background(), engine.SessionConfig{
		AvailableProbeServices: []model.OOAPIService{{
			Address: "https://ams-pg-test.ooni.org",
			Type:    "https",
		}},
		Logger:          log.Log,
		SoftwareName:    "ooniprobe-engine",
		SoftwareVersion: "0.0.1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if lookupBackends {
		if err := sess.MaybeLookupBackends(); err != nil {
			t.Fatal(err)
		}
	}
	if err := sess.MaybeLookupLocation(); err != nil {
		t.Fatal(err)
	}
	return sess
}
