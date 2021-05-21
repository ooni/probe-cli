package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/netplumbing"
	utls "gitlab.com/yawning/utls.git"
)

/*
	Bottom line: this client can leverage existing Go functionality
	to easily connect to either socks5 or http proxy w/ auth.
*/

func main() {
	log.SetLevel(log.DebugLevel)
	txp := netplumbing.DefaultTransport
	utlsHandshaker := &netplumbing.UTLSHandshaker{
		ClientHelloID: &utls.HelloChrome_Auto,
	}
	config := &netplumbing.Config{
		Logger:        log.Log,
		TLSHandshaker: utlsHandshaker,
		/*
			Proxy: &url.URL{
				//Scheme: "socks5",
				Scheme: "http",
				//Host: "127.0.0.1:9050",
				//Host: "127.0.0.1:8118",
				Host: "127.0.0.1:8002",
				User: url.UserPassword("antani", "mascetti"),
			},
		*/
		//HTTPTransport: txp.HTTP3RoundTripper,
		HTTPTransport: txp.OORoundTripper,
	}
	ctx := netplumbing.WithConfig(context.Background(), config)
	clnt := &http.Client{Transport: txp}
	get(ctx, clnt, "http://nexa.polito.it")
	get(ctx, clnt, "http://nexa.polito.it/robots.txt")
}

func get(ctx context.Context, clnt *http.Client, url string) {
	theader := &netplumbing.TraceHeader{}
	ctx = netplumbing.WithTraceHeader(ctx, theader)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		log.WithError(err).Fatal("http.NewRequest failed")
	}
	resp, err := clnt.Do(req)
	if err != nil {
		log.WithError(err).Fatal("clnt.Get failed")
	}
	defer resp.Body.Close()
	data, err := netplumbing.ReadAllContext(ctx, resp.Body)
	if err != nil {
		log.WithError(err).Fatal("ioutil.ReadAll failed")
	}
	log.Infof("got %d bytes", len(data))
	for _, ev := range theader.MoveOut() {
		data, _ := json.Marshal(map[string]interface{}{ev.Kind(): ev})
		fmt.Printf("%s\n", string(data))
	}
	fmt.Printf("\"separator\"\n")
}
