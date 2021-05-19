package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/netplumbing"
)

/*
	Bottom line: this client can leverage existing Go functionality
	to easily connect to either socks5 or http proxy w/ auth.

	YET, PROBLEM: why don't I see the TLS handshake when I am using
	the proxy in this mode? What is happening? Trace:

2021/05/19 20:01:46 debug > GET https://www.google.com
2021/05/19 20:01:46 debug >
2021/05/19 20:01:46 debug http: using proxy: http://127.0.0.1:8118
2021/05/19 20:01:46 debug dial: 127.0.0.1:8118/tcp...
2021/05/19 20:01:46 debug connect: 127.0.0.1:8118/tcp...
2021/05/19 20:01:46 debug connect: 127.0.0.1:8118/tcp... ok
2021/05/19 20:01:46 debug dial: 127.0.0.1:8118/tcp... ok
2021/05/19 20:01:50 debug < 200

	This is very annoying because it's breaking the way
	in which the psiphon code typically works.
*/

func main() {
	log.SetLevel(log.DebugLevel)
	config := &netplumbing.Config{
		Logger: log.Log,
		Proxy: &url.URL{
			Scheme: "socks5",
			//Scheme: "http",
			Host: "127.0.0.1:9050",
			//Host: "127.0.0.1:8118",
		},
	}
	ctx := netplumbing.WithConfig(context.Background(), config)
	theader := &netplumbing.TraceHeader{}
	ctx = netplumbing.WithTraceHeader(ctx, theader)
	req, err := http.NewRequestWithContext(ctx, "GET", "https://www.google.com", nil)
	if err != nil {
		log.WithError(err).Fatal("http.NewRequest failed")
	}
	resp, err := netplumbing.DefaultTransport.RoundTrip(req)
	if err != nil {
		log.WithError(err).Fatal("clnt.Get failed")
	}
	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.WithError(err).Fatal("ioutil.ReadAll failed")
	}
	log.Infof("got %d bytes", len(data))
	for _, ev := range theader.MoveOut() {
		data, _ := json.Marshal(ev)
		fmt.Printf("%s\n", string(data))
	}
}
