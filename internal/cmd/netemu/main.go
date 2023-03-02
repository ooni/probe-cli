package main

//
// Test client for ./internal/qa.
//
// Will be removed before merging to master.
//

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

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
	defer env.Close()
	gginfo := env.NonCensoredStaticGetaddrinfo()

	ctx := context.Background()

	if *index == 0 || *index == 1 {
		fmt.Fprintf(os.Stderr, "\n\n\n")
		log.Infof("WITH THE FASTEST LINK")
		_, err := env.RunExperiment(ctx, env.NewUNetStack(gginfo), &netem.LinkConfig{})
		log.Infof("ERROR: %+v", err)
		fmt.Fprintf(os.Stderr, "\n\n\n")
	}

	if *index == 0 || *index == 2 {
		fmt.Fprintf(os.Stderr, "\n\n\n")
		log.Infof("WITH THE MEDIUM LINK")
		linkConfig := &netem.LinkConfig{
			Dump:             false,
			LeftToRightPLR:   0.00001,
			LeftToRightDelay: 5 * time.Millisecond,
			RightToLeftDelay: 5 * time.Millisecond,
			RightToLeftPLR:   0.00001,
		}
		_, err := env.RunExperiment(ctx, env.NewUNetStack(gginfo), linkConfig)
		log.Infof("ERROR: %+v", err)
		fmt.Fprintf(os.Stderr, "\n\n\n")
	}

	if *index == 0 || *index == 3 {
		fmt.Fprintf(os.Stderr, "\n\n\n")
		log.Infof("WITH THE SLOWEST LINK")
		linkConfig := &netem.LinkConfig{
			Dump:             false,
			LeftToRightPLR:   0,
			LeftToRightDelay: 100 * time.Millisecond,
			RightToLeftDelay: 100 * time.Millisecond,
			RightToLeftPLR:   0.1,
		}
		_, err := env.RunExperiment(ctx, env.NewUNetStack(gginfo), linkConfig)
		log.Infof("ERROR: %+v", err)
		fmt.Fprintf(os.Stderr, "\n\n\n")
	}

	if *index == 0 || *index == 4 {
		fmt.Fprintf(os.Stderr, "\n\n\n")
		log.Infof("WITH DPI DROPPING TRAFFIC TO DASH SERVER")
		dpi := &netem.DPIDropTrafficForServerEndpoint{
			ServerIPAddress: env.DASHServerIPAddress(),
			ServerPort:      443,
			ServerProtocol:  layers.IPProtocolTCP,
			DPIStack:        env.NewUNetStack(gginfo),
		}
		_, err := env.RunExperiment(ctx, dpi, &netem.LinkConfig{})
		log.Infof("ERROR: %+v", err)
		fmt.Fprintf(os.Stderr, "\n\n\n")
	}

	if *index == 0 || *index == 5 {
		fmt.Fprintf(os.Stderr, "\n\n\n")
		log.Infof("WITH DPI DROPPING TRAFFIC FOR MLAB-NS")
		dpi := &netem.DPIDropTrafficForServerEndpoint{
			ServerIPAddress: env.MLabLocateServerIPAddress(),
			ServerPort:      443,
			ServerProtocol:  layers.IPProtocolTCP,
			DPIStack:        env.NewUNetStack(gginfo),
		}
		_, err := env.RunExperiment(ctx, dpi, &netem.LinkConfig{})
		log.Infof("ERROR: %+v", err)
		fmt.Fprintf(os.Stderr, "\n\n\n")
	}

	if *index == 0 || *index == 6 {
		fmt.Fprintf(os.Stderr, "\n\n\n")
		log.Infof("WITH DPI DROPPING TRAFFIC FOR DASH SNI")
		dpi := netem.NewDPIDropTrafficForTLSSNI(
			env.NewUNetStack(gginfo),
			env.DASHServerDomainName(),
		)
		_, err := env.RunExperiment(ctx, dpi, &netem.LinkConfig{})
		log.Infof("ERROR: %+v", err)
		fmt.Fprintf(os.Stderr, "\n\n\n")
	}

	if *index == 0 || *index == 7 {
		fmt.Fprintf(os.Stderr, "\n\n\n")
		log.Infof("WITH DPI THROTTLING TRAFFIC FOR DASH SNI")
		dpi := netem.NewDPIThrottleTrafficForTLSSNI(
			env.NewUNetStack(gginfo),
			env.DASHServerDomainName(),
			0.19,
		)
		linkConfig := &netem.LinkConfig{
			Dump:             false,
			LeftToRightPLR:   0,
			LeftToRightDelay: 30 * time.Millisecond,
			RightToLeftDelay: 30 * time.Millisecond,
			RightToLeftPLR:   0,
		}
		_, err := env.RunExperiment(ctx, dpi, linkConfig)
		log.Infof("ERROR: %+v", err)
		fmt.Fprintf(os.Stderr, "\n\n\n")
	}
}
