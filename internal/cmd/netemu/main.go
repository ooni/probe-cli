package main

//
// Test client for ./internal/qa.
//
// Will be removed before merging to master.
//

import (
	"flag"
	"fmt"
	"os"

	"github.com/apex/log"
	"github.com/google/gopacket/layers"
	"github.com/ooni/probe-cli/v3/internal/logx"
	"github.com/ooni/probe-cli/v3/internal/netem"
	"github.com/ooni/probe-cli/v3/internal/qa"
)

func main() {
	index := flag.Int64("index", 0, "index of the test to run")
	flag.Parse()

	logHandler := logx.NewHandlerWithDefaultSettings()
	logHandler.Emoji = true
	log.Log = &log.Logger{Level: log.InfoLevel, Handler: logHandler}

	env := qa.NewDASHEnvironment()
	defer env.Stop()
	gginfo := env.NonCensoredStaticGetaddrinfo()

	if *index == 0 || *index == 1 {
		fmt.Fprintf(os.Stderr, "\n\n\n")
		log.Infof("WITH THE FASTEST LINK")
		linkFactory := netem.NewLinkFastest
		dpi := &netem.DPINone{}
		_, err := env.RunExperiment(gginfo, linkFactory, dpi)
		log.Infof("ERROR: %+v", err)
		fmt.Fprintf(os.Stderr, "\n\n\n")
	}

	if *index == 0 || *index == 2 {
		fmt.Fprintf(os.Stderr, "\n\n\n")
		log.Infof("WITH THE MEDIUM LINK")
		linkFactory := netem.NewLinkMedium
		dpi := &netem.DPINone{}
		_, err := env.RunExperiment(gginfo, linkFactory, dpi)
		log.Infof("ERROR: %+v", err)
		fmt.Fprintf(os.Stderr, "\n\n\n")
	}

	if *index == 0 || *index == 3 {
		fmt.Fprintf(os.Stderr, "\n\n\n")
		log.Infof("WITH THE SLOWEST LINK")
		linkFactory := netem.NewLinkSlowest
		dpi := &netem.DPINone{}
		_, err := env.RunExperiment(gginfo, linkFactory, dpi)
		log.Infof("ERROR: %+v", err)
		fmt.Fprintf(os.Stderr, "\n\n\n")
	}

	if *index == 0 || *index == 4 {
		fmt.Fprintf(os.Stderr, "\n\n\n")
		log.Infof("WITH DPI DROPPING TRAFFIC TO DASH SERVER")
		linkFactory := netem.NewLinkFastest
		dpi := &netem.DPIDropTrafficForServerEndpoint{
			Direction:       netem.LinkDirectionLeftToRight,
			ServerIPAddress: env.DASHServerIPAddress(),
			ServerPort:      443,
			ServerProtocol:  layers.IPProtocolTCP,
		}
		_, err := env.RunExperiment(gginfo, linkFactory, dpi)
		log.Infof("ERROR: %+v", err)
		fmt.Fprintf(os.Stderr, "\n\n\n")
	}

	if *index == 0 || *index == 5 {
		fmt.Fprintf(os.Stderr, "\n\n\n")
		log.Infof("WITH DPI DROPPING TRAFFIC FOR MLAB-NS")
		linkFactory := netem.NewLinkFastest
		dpi := &netem.DPIDropTrafficForServerEndpoint{
			Direction:       netem.LinkDirectionLeftToRight,
			ServerIPAddress: env.MLabLocateServerIPAddress(),
			ServerPort:      443,
			ServerProtocol:  layers.IPProtocolTCP,
		}
		_, err := env.RunExperiment(gginfo, linkFactory, dpi)
		log.Infof("ERROR: %+v", err)
		fmt.Fprintf(os.Stderr, "\n\n\n")
	}

	if *index == 0 || *index == 6 {
		fmt.Fprintf(os.Stderr, "\n\n\n")
		log.Infof("WITH DPI DROPPING TRAFFIC FOR DASH SNI")
		linkFactory := netem.NewLinkFastest
		dpi := netem.NewDPIDropTrafficForTLSSNI(env.DASHServerDomainName())
		_, err := env.RunExperiment(gginfo, linkFactory, dpi)
		log.Infof("ERROR: %+v", err)
		fmt.Fprintf(os.Stderr, "\n\n\n")
	}

	if *index == 0 || *index == 7 {
		fmt.Fprintf(os.Stderr, "\n\n\n")
		log.Infof("WITH DPI THROTTLING TRAFFIC FOR DASH SNI")
		linkFactory := netem.NewLinkFastest
		dpi := netem.NewDPIThrottleTrafficForTLSSNI(env.DASHServerDomainName())
		_, err := env.RunExperiment(gginfo, linkFactory, dpi)
		log.Infof("ERROR: %+v", err)
		fmt.Fprintf(os.Stderr, "\n\n\n")
	}
}
