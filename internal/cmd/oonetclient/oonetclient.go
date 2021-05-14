package main

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/oonet"
)

/*
	Bottom line: this client can leverage existing Go functionality
	to easily connect to either socks5 or http proxy w/ auth.
*/

func main() {
	log.SetLevel(log.DebugLevel)
	txp := &oonet.Transport{Logger: log.Log}
	clnt := &http.Client{Transport: txp}
	ctx := context.Background()
	ctx = oonet.WithOverrides(ctx, &oonet.Overrides{
		Proxy: &url.URL{
			Scheme: "http",
			User:   url.UserPassword("antani", "melandri"),
			Host:   "127.0.0.1:8002",
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
