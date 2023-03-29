package nettests

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/cmd/ooniprobe/internal/ooni"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/platform"
	"github.com/pkg/errors"
)

// RunGroupConfig contains the settings for running a nettest group.
type RunGroupConfig struct {
	GroupName  string
	InputFiles []string
	Inputs     []string
	Probe      *ooni.Probe
	RunType    model.RunType // hint for check-in API
}

const websitesURLLimitRemoved = `WARNING: CONFIGURATION CHANGE REQUIRED:

* Since ooniprobe 3.9.0, websites_url_limit has been replaced
  by websites_max_runtime in the configuration

* To silence this warning either set websites_url_limit to zero or
  replace it with websites_max_runtime

* For the rest of 2021, we will automatically convert websites_url_limit
  to websites_max_runtime (if the latter is not already set)

* We will consider that each URL in websites_url_limit takes five
  seconds to run and thus calculate websites_max_runtime

* Since 2022, we will start silently ignoring websites_url_limit
`

var deprecationWarningOnce sync.Once

// RunGroup runs a group of nettests according to the specified config.
func RunGroup(config RunGroupConfig) error {
	if config.Probe.Config().Nettests.WebsitesURLLimit > 0 {
		if config.Probe.Config().Nettests.WebsitesMaxRuntime <= 0 {
			limit := config.Probe.Config().Nettests.WebsitesURLLimit
			maxRuntime := 5 * limit
			config.Probe.Config().Nettests.WebsitesMaxRuntime = maxRuntime
		}
		deprecationWarningOnce.Do(func() {
			log.Warn(websitesURLLimitRemoved)
			time.Sleep(30 * time.Second)
		})
	}

	if config.Probe.IsTerminated() {
		log.Debugf("context is terminated, stopping runNettestGroup early")
		return nil
	}

	sess := config.Probe.NewSession(context.Background(), config.RunType)
	defer sess.Close()

	if err := sess.Bootstrap(context.Background()); err != nil {
		log.WithError(err).Error("Failed to bootstrap the measurement session")
		return err
	}

	location, err := sess.Geolocate(context.Background())
	if err != nil {
		log.WithError(err).Error("Failed to lookup the location of the probe")
		return err
	}
	db := config.Probe.DB()
	network, err := db.CreateNetwork(location)
	if err != nil {
		log.WithError(err).Error("Failed to create the network row")
		return err
	}

	log.Debugf(
		"Enabled category codes are the following %v",
		config.Probe.Config().Nettests.WebsitesEnabledCategoryCodes,
	)
	checkInConfig := &model.OOAPICheckInConfig{
		// Setting Charging and OnWiFi to true causes the CheckIn
		// API to return to us as much URL as possible with the
		// given RunType hint.
		Charging:        true,
		OnWiFi:          true,
		Platform:        platform.Name(),
		ProbeASN:        location.ProbeASNString(),
		ProbeCC:         location.ProbeCC(),
		RunType:         config.RunType,
		SoftwareName:    sess.BootstrapRequest().SoftwareName,
		SoftwareVersion: sess.BootstrapRequest().SoftwareVersion,
		WebConnectivity: model.OOAPICheckInConfigWebConnectivity{
			CategoryCodes: config.Probe.Config().Nettests.WebsitesEnabledCategoryCodes,
		},
	}
	if checkInConfig.WebConnectivity.CategoryCodes == nil {
		checkInConfig.WebConnectivity.CategoryCodes = []string{}
	}
	checkInResult, err := sess.CheckIn(context.Background(), checkInConfig)
	if err != nil {
		log.WithError(err).Warn("Failed to query the check-in API")
		return err
	}

	group, ok := All[config.GroupName]
	if !ok {
		log.Errorf("No test group named %s", config.GroupName)
		return errors.New("invalid test group name")
	}
	log.Debugf("Running test group %s", group.Label)

	result, err := db.CreateResult(
		config.Probe.Home(), config.GroupName, network.ID)
	if err != nil {
		log.Errorf("DB result error: %s", err)
		return err
	}

	config.Probe.ListenForSignals()
	config.Probe.MaybeListenForStdinClosed()
	for i, nt := range group.Nettests {
		if config.Probe.IsTerminated() {
			log.Debugf("context is terminated, stopping group.Nettests early")
			break
		}
		if config.RunType != model.RunTypeTimed {
			if _, background := nt.(onlyBackground); background {
				log.Debug("we only run this nettest in background mode")
				continue
			}
		}
		log.Debugf("Running test %T", nt)
		ctl := NewController(nt, config.Probe, result, sess)
		ctl.CheckInResult = checkInResult
		ctl.InputFiles = config.InputFiles
		ctl.Inputs = config.Inputs
		ctl.RunType = config.RunType
		ctl.SetNettestIndex(i, len(group.Nettests))
		if err = nt.Run(ctl); err != nil {
			log.WithError(err).Errorf("Failed to run %s", group.Label)
		}
	}

	// Remove the directory if it's emtpy, which happens when the corresponding
	// measurements have been submitted (see https://github.com/ooni/probe/issues/2090)
	dir, err := os.Open(result.MeasurementDir)
	if err != nil {
		return err
	}
	defer dir.Close()
	_, err = dir.Readdirnames(1)
	if err != nil {
		os.Remove(result.MeasurementDir)
	}
	if err = db.Finished(result); err != nil {
		return err
	}
	return nil
}

// onlyBackground is the interface implements by nettests that we don't
// want to run in manual mode because they take too much runtime
//
// See:
//
// - https://github.com/ooni/probe/issues/2101
//
// - https://github.com/ooni/probe/issues/2057
type onlyBackground interface {
	onlyBackground()
}
