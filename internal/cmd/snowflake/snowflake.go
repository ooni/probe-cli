package main

import (
	"context"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/engine/tunnel"
)

func main() {
	log.SetLevel(log.DebugLevel)
	sfk := &tunnel.Snowflake{
		BrokerURL:        "https://snowflake-broker.torproject.net.global.prod.fastly.net/",
		Capacity:         3,
		FrontDomain:      "cdn.sstatic.net",
		ICEServersCommas: "stun:stun.voip.blackberry.com:3478,stun:stun.altar.com.pl:3478,stun:stun.antisip.com:3478,stun:stun.bluesip.net:3478,stun:stun.dus.net:3478,stun:stun.epygi.com:3478,stun:stun.sonetel.com:3478,stun:stun.sonetel.net:3478,stun:stun.stunprotocol.org:3478,stun:stun.uls.co.za:3478,stun:stun.voipgate.com:3478,stun:stun.voys.nl:3478",
		Logger:           log.Log,
	}
	sfk.Start(context.Background())
	<-context.Background().Done()
}
