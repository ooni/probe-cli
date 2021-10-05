// Command jafar2 implements censorship policies.
package main

import (
	"flag"

	"github.com/apex/log"
)

func main() {
	defer FatalOnPanic()

	configFile := flag.String("f", "", "config file")
	keep := flag.Bool("k", false, "keep temporary files")
	verbose := flag.Bool("v", false, "verbose mode")
	flag.Parse()
	log.SetHandler(NewLoggerHandler())
	if *verbose {
		log.SetLevel(log.DebugLevel)
	}
	if *configFile == "" {
		log.Fatal("usage: go run ./internal/cmd/jafar2 -f <config.json>")
	}
	sh := NewLinuxShell()

	log.Infof("reading config file: %s...", *configFile)
	conf := NewConfig(*configFile)
	log.Infof("reading config file: %s... ok", *configFile)

	log.Info("recompiling miniooni...")
	miniooni := NewMiniooni(sh)
	log.Infof("recompiling miniooni... %s", miniooni.Path())
	if !*keep {
		defer func() {
			log.Infof("removing miniooni... %s", miniooni.Path())
			miniooni.Cleanup()
			log.Info("removing miniooni... ok")
		}()
	}

	log.Info("writing trampoline script...")
	trampoline := NewTrampoline(conf, miniooni)
	log.Infof("writing trampoline script... %s", trampoline.Path())
	if !*keep {
		defer func() {
			log.Infof("removing trampoline script... %s", trampoline.Path())
			trampoline.Cleanup()
			log.Info("removing trampoline script... ok")
		}()
	}

	log.Info("rebuilding container...")
	di := NewDockerImage(conf, sh)
	log.Infof("rebuilding container... %s", di.Name())
	if !*keep {
		defer func() {
			log.Info("removing docker container build dir...")
			di.Cleanup()
			log.Info("removing docker container build dir... ok")
		}()
	}

	log.Info("creating docker network...")
	dn := NewDockerNetwork(conf, sh)
	log.Infof("creating docker network... name=%s, bridge=%s", dn.Name(), dn.Bridge())
	defer func() {
		log.Infof("removing docker network... %s", dn.Name())
		dn.Cleanup()
		log.Info("removing docker network... ok")
	}()

	log.Info("adding iptables blackholing rules...")
	iptables := NewIPTablesBlockingPolicies(conf, dn.Bridge(), sh)
	log.Info("adding iptables blackholing rules... ok")
	defer func() {
		log.Info("removing iptables blackholing...")
		iptables.Waive()
		log.Info("removing iptables blackholing... ok")
	}()

	DockerRun(conf, sh, dn.Bridge(), dn.Name(), di.Name(), trampoline.Path())
}
