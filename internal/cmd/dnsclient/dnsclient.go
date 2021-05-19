package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/apex/log"
	"github.com/miekg/dns"
	"github.com/ooni/probe-cli/v3/internal/netplumbing"
)

func main() {
	log.SetLevel(log.DebugLevel)

	question := dns.Question{
		Name:   dns.Fqdn("www.youtube.com"),
		Qtype:  dns.TypeA,
		Qclass: dns.ClassINET,
	}
	query := &dns.Msg{}
	query.Id = dns.Id()
	query.RecursionDesired = true
	query.Question = make([]dns.Question, 1)
	query.Question[0] = question

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

	for _, ev := range theader.MoveOut() {
		data, _ := json.Marshal(map[string]interface{}{ev.Kind(): ev})
		fmt.Printf("%s\n", string(data))
	}
}
