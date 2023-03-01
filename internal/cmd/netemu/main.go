package main

//
// Test client for ./internal/qa.
//
// Will be removed before merging to master.
//

import (
	"log"

	"github.com/google/gopacket/layers"
	"github.com/ooni/probe-cli/v3/internal/netem"
	"github.com/ooni/probe-cli/v3/internal/qa"
)

func main() {
	env := qa.NewDASHEnvironment()
	defer env.Stop()
	gginfo := env.NonCensoredStaticGetaddrinfo()

	if true {
		linkFactory := netem.NewLinkFastest
		dpi := &netem.DPINone{}
		_, err := env.RunExperiment(gginfo, linkFactory, dpi)
		log.Printf("ERROR: %+v", err)
	}

	if false {
		linkFactory := netem.NewLinkMedium
		dpi := &netem.DPINone{}
		_, err := env.RunExperiment(gginfo, linkFactory, dpi)
		log.Printf("ERROR: %+v", err)
	}

	if false {
		linkFactory := netem.NewLinkSlowest
		dpi := &netem.DPINone{}
		_, err := env.RunExperiment(gginfo, linkFactory, dpi)
		log.Printf("ERROR: %+v", err)
	}

	if false {
		linkFactory := netem.NewLinkFastest
		dpi := &netem.DPIDropTrafficForServerEndpoint{
			Direction:       netem.LinkDirectionLeftToRight,
			ServerIPAddress: env.DASHServerIPAddress(),
			ServerPort:      443,
			ServerProtocol:  layers.IPProtocolTCP,
		}
		_, err := env.RunExperiment(gginfo, linkFactory, dpi)
		log.Printf("ERROR: %+v", err)
	}

	if false {
		linkFactory := netem.NewLinkFastest
		dpi := &netem.DPIDropTrafficForServerEndpoint{
			Direction:       netem.LinkDirectionLeftToRight,
			ServerIPAddress: env.MLabLocateServerIPAddress(),
			ServerPort:      443,
			ServerProtocol:  layers.IPProtocolTCP,
		}
		_, err := env.RunExperiment(gginfo, linkFactory, dpi)
		log.Printf("ERROR: %+v", err)
	}

	if false {
		linkFactory := netem.NewLinkFastest
		dpi := netem.NewDPIDropTrafficForTLSSNI(env.DASHServerDomainName())
		_, err := env.RunExperiment(gginfo, linkFactory, dpi)
		log.Printf("ERROR: %+v", err)
	}

	if false {
		linkFactory := netem.NewLinkFastest
		dpi := netem.NewDPIThrottleTrafficForTLSSNI(env.DASHServerDomainName())
		_, err := env.RunExperiment(gginfo, linkFactory, dpi)
		log.Printf("ERROR: %+v", err)
	}
}
