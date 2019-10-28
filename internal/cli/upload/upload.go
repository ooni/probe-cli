package upload

import (
	"encoding/json"
	"errors"

	"github.com/alecthomas/kingpin"
	"github.com/apex/log"
	"github.com/ooni/probe-cli/internal/cli/root"
	"github.com/ooni/probe-cli/internal/database"
)

func init() {
	cmd := root.Command("upload", "Upload a specific measurement")
	msmtID := cmd.Arg(
		"id", "the id of the measurement to upload",
	).Required().Int64()

	cmd.Action(func(_ *kingpin.ParseContext) error {

		// Step 1: init, make sure user is informed, and make sure that
		// we have the permission to upload measurements
		ctx, err := root.Init()
		if err != nil {
			log.Errorf("%s", err)
			return err
		}
		if err = ctx.MaybeOnboarding(); err != nil {
			log.WithError(err).Error("failed to perform onboarding")
			return err
		}
		if ctx.Config.Sharing.UploadResults == false {
			log.Error("upload is disabled in the settings")
			return errors.New("upload disabled")
		}

		// Step 2: we need testName and measurement bytes
		//
		// Because of the way in which tests are stored and measurements are
		// searched, we need to use this function. I believe we may be happier
		// without using JSONL, but we cannot do this now.
		//
		// This is to say, it is a bit sad that we need to marshal again the
		// piece of data, yet probe-engine wants a measurement to parse.
		msmt, err := database.GetMeasurementJSON(ctx.DB, *msmtID)
		if err != nil {
			log.Errorf("error: %v", err)
			return err
		}
		testName, ok := msmt["test_name"].(string)
		if !ok {
			log.Error("this does not seem to be a valid measurement")
			return errors.New("invalid measurement")
		}
		data, err := json.Marshal(msmt)
		if err != nil {
			log.Errorf("error: %v", err)
			return err
		}

		// Step 3: configure the session properly to have an experiment
		if ctx.Config.Advanced.BouncerURL != "" {
			log.Debugf("Using bouncer: %s", ctx.Config.Advanced.CollectorURL)
			ctx.Session.AddAvailableHTTPSBouncer(ctx.Config.Advanced.BouncerURL)
		}
		if ctx.Config.Advanced.CollectorURL != "" {
			log.Debugf("Using collector: %s", ctx.Config.Advanced.CollectorURL)
			ctx.Session.AddAvailableHTTPSCollector(ctx.Config.Advanced.CollectorURL)
		} else if err := ctx.Session.MaybeLookupBackends(); err != nil {
			log.WithError(err).Warn("Failed to discover OONI backends")
			return err
		}
		// Make sure we share what the user wants us to share.
		ctx.Session.SetIncludeProbeIP(ctx.Config.Sharing.IncludeIP)
		ctx.Session.SetIncludeProbeASN(ctx.Config.Sharing.IncludeASN)
		ctx.Session.SetIncludeProbeCC(ctx.Config.Sharing.IncludeCountry)

		// Step 4: load measurement as part of experiment
		builder, err := ctx.Session.NewExperimentBuilder(testName)
		if err != nil {
			log.Errorf("error: %v", err)
			return err
		}
		experiment := builder.Build()
		measurement, err := experiment.LoadMeasurement(data)
		if err != nil {
			log.Errorf("error: %v", err)
			return err
		}

		// Step 5: run the measurement submission flow
		err = experiment.OpenReport()
		if err != nil {
			log.Errorf("error: %v", err)
			return err
		}
		defer experiment.CloseReport()
		err = experiment.SubmitAndUpdateMeasurement(measurement)
		if err != nil {
			log.Errorf("error: %v", err)
			return err
		}

		// Step 6: write back into the database
		// ???

		return nil
	})
}
