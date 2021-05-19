package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/netplumbing"
)

func main() {
	log.SetLevel(log.DebugLevel)

	query := netplumbing.DefaultTransport.DNSEncodeA("www.youtube.com", true)

	resolverURL := &url.URL{
		Scheme: "https",
		Host:   "8.8.8.8",
		Path:   "/dns-query",
	}

	config := &netplumbing.Config{
		Logger: log.Log,
		//HTTPTransport: netplumbing.DefaultTransport.HTTP3RoundTripper,
	}
	ctx := netplumbing.WithConfig(context.Background(), config)
	theader := &netplumbing.TraceHeader{}
	ctx = netplumbing.WithTraceHeader(ctx, theader)

	reply, err := netplumbing.DefaultTransport.DNSQuery(ctx, resolverURL, query)
	if err != nil {
		log.WithError(err).Fatal("cannot send query")
	}

	log.Infof("reply: %s", reply)

	addrs, err := netplumbing.DefaultTransport.DNSDecodeA(reply)
	if err != nil {
		log.WithError(err).Fatal("cannot decode reply")
	}
	for _, addr := range addrs {
		log.Infof("- addr: %s", addr)
	}

	for _, ev := range theader.MoveOut() {
		data, _ := json.Marshal(map[string]interface{}{ev.Kind(): ev})
		fmt.Printf("%s\n", string(data))
	}
}
