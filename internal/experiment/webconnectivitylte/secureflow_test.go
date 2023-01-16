package webconnectivity

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/atomicx"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func TestSecureFlow_Run(t *testing.T) {
	type fields struct {
		Address         string
		DNSCache        *DNSCache
		IDGenerator     *atomicx.Int64
		Logger          model.Logger
		NumRedirects    *NumRedirects
		TestKeys        *TestKeys
		ZeroTime        time.Time
		WaitGroup       *sync.WaitGroup
		ALPN            []string
		CookieJar       http.CookieJar
		FollowRedirects bool
		HostHeader      string
		PrioSelector    *prioritySelector
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
		want: errNotAllowedToConnect,
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
		want: errNotAllowedToConnect,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &SecureFlow{
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
