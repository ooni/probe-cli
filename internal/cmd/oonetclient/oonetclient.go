package main

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/netplumbing"
)

/*
	Bottom line: this client can leverage existing Go functionality
	to easily connect to either socks5 or http proxy w/ auth.
*/

func main() {
	log.SetLevel(log.DebugLevel)
	clnt := &http.Client{Transport: netplumbing.DefaultTransport}
	ctx := netplumbing.WithSettings(context.Background(), &netplumbing.Settings{
		Logger: log.Log,
		Proxy: &url.URL{
			Scheme: "socks5",
			Host:   "127.0.0.1:9050",
		},
	})
	req, err := http.NewRequestWithContext(ctx, "GET", "https://www.google.com", nil)
	if err != nil {
		log.WithError(err).Fatal("http.NewRequest failed")
	}
	resp, err := clnt.Do(req)
	if err != nil {
		log.WithError(err).Fatal("clnt.Get failed")
	}
	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.WithError(err).Fatal("ioutil.ReadAll failed")
	}
	log.Infof("got %d bytes", len(data))
}
