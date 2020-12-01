package nettests

import (
	"github.com/apex/log"
	"github.com/ooni/probe-cli/internal/database"
	"github.com/ooni/probe-cli/internal/ooni"
	"github.com/pkg/errors"
)

// RunGroupConfig contains the settings for running a nettest group.
type RunGroupConfig struct {
	GroupName  string
	Probe      *ooni.Probe
	InputFiles []string
	Inputs     []string
}

// RunGroup runs a group of nettests according to the specified config.
func RunGroup(config RunGroupConfig) error {
	if config.Probe.IsTerminated() == true {
		log.Debugf("context is terminated, stopping runNettestGroup early")
		return nil
	}

	sess, err := config.Probe.NewSession()
	if err != nil {
		log.WithError(err).Error("Failed to create a measurement session")
		return err
	}
	defer sess.Close()

	err = sess.MaybeLookupLocation()
	if err != nil {
		log.WithError(err).Error("Failed to lookup the location of the probe")
		return err
	}
	network, err := database.CreateNetwork(config.Probe.DB(), sess)
	if err != nil {
		log.WithError(err).Error("Failed to create the network row")
		return err
	}
	if err := sess.MaybeLookupBackends(); err != nil {
		log.WithError(err).Warn("Failed to discover OONI backends")
		return err
	}

	group, ok := NettestGroups[config.GroupName]
	if !ok {
		log.Errorf("No test group named %s", config.GroupName)
		return errors.New("invalid test group name")
	}
	log.Debugf("Running test group %s", group.Label)

	result, err := database.CreateResult(
		config.Probe.DB(), config.Probe.Home(), config.GroupName, network.ID)
	if err != nil {
		log.Errorf("DB result error: %s", err)
		return err
	}

	config.Probe.ListenForSignals()
	config.Probe.MaybeListenForStdinClosed()
	for i, nt := range group.Nettests {
		if config.Probe.IsTerminated() == true {
			log.Debugf("context is terminated, stopping group.Nettests early")
			break
		}
		log.Debugf("Running test %T", nt)
		ctl := NewController(nt, config.Probe, result, sess)
		ctl.InputFiles = config.InputFiles
		ctl.Inputs = config.Inputs
		ctl.SetNettestIndex(i, len(group.Nettests))
		if err = nt.Run(ctl); err != nil {
			log.WithError(err).Errorf("Failed to run %s", group.Label)
		}
	}

	if err = result.Finished(config.Probe.DB()); err != nil {
		return err
	}
	return nil
}
