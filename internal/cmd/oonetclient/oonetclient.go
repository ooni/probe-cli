package main

import (
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
	txp := &oonet.Transport{
		Logger: log.Log,
		Proxy: func(*http.Request) (*url.URL, error) {
			/*
				return &url.URL{
					Scheme: "socks5",
					User:   url.UserPassword("antani", "melandri"),
					Host:   "127.0.0.1:8118",
				}, nil
			*/
			return &url.URL{
				Scheme: "http",
				User:   url.UserPassword("antani", "melandri"),
				Host:   "127.0.0.1:8002",
			}, nil
		},
	}
	clnt := &http.Client{Transport: txp}
	resp, err := clnt.Get("https://www.google.com")
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
