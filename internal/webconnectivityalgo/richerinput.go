package webconnectivityalgo

//
// Richer-input support for Web Connectivity v0.4 and v0.5
//

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/richerinput"
)

// NewRicherInputExperiment constructs a new [model.RicherInputExperiment].
func NewRicherInputExperiment(sess model.RicherInputSession, mx model.ExperimentMeasurer) model.RicherInputExperiment {
	return richerinput.NewExperiment(
		&richerInputLoader{
			sess: sess,
		},
		&richerInputMeasurer{
			m: mx,
		},
		sess,
	)
}

// richerInputTarget is the kind of richer input used by this experiment.
type richerInputTarget struct {
	info model.OOAPIURLInfo
}

// CategoryCode implements [model.RicherInputTarget].
func (r richerInputTarget) CategoryCode() string {
	return r.info.CategoryCode
}

// CountryCode implements [model.RicherInputTarget].
func (r richerInputTarget) CountryCode() string {
	return r.info.CountryCode
}

// Options implements [model.RicherInputTarget].
func (r richerInputTarget) Options() []string {
	return nil
}

// Input implements [model.RicherInputTarget].
func (r richerInputTarget) Input() model.MeasurementInput {
	return model.MeasurementInput(r.info.URL)
}

// richerInputLoader loads richer input for this experiment.
type richerInputLoader struct {
	sess model.RicherInputSession
}

// Load implements InputLoader.
func (v *richerInputLoader) Load(ctx context.Context, config *model.RicherInputConfig) ([]richerInputTarget, error) {
	// Handle the case where the user did not supply any input.
	if !config.ContainsUserConfiguredInput() {
		return v.callRicherInputAPI(ctx, config)
	}

	// Read Inputs and InputFilePaths.
	inputs, err := richerinput.LoadInputs(config)
	if err != nil {
		return nil, err
	}

	// Compute the product of the options and the inputs.
	var product []richerInputTarget
	for _, input := range inputs {
		product = append(product, richerInputTarget{
			info: model.OOAPIURLInfo{
				CategoryCode: model.DefaultCategoryCode,
				CountryCode:  model.DefaultCountryCode,
				URL:          input,
			},
		})
	}

	return product, nil
}

// callRicherInputAPI invokes the richer-input API for this experiment.
func (v *richerInputLoader) callRicherInputAPI(ctx context.Context, _ *model.RicherInputConfig) ([]richerInputTarget, error) {
	// TODO(bassosimone): we should add additional fields inside the richer input config
	// such that we're able to properly invoke the check-in API.
	//
	// For now, many of the fields we're using here are like default stub values.
	req := &model.OOAPICheckInConfig{
		Charging:        false, // TODO(bassosimone): properly fill
		OnWiFi:          false, // TODO(bassosimone): property fill
		Platform:        v.sess.Platform(),
		ProbeASN:        v.sess.ProbeASNString(),
		ProbeCC:         v.sess.ProbeCC(),
		RunType:         model.RunTypeManual, // TODO(bassosimone): property fill
		SoftwareName:    v.sess.SoftwareName(),
		SoftwareVersion: v.sess.SoftwareVersion(),
		WebConnectivity: model.OOAPICheckInConfigWebConnectivity{
			CategoryCodes: []string{}, // TODO(bassosimone): property fill
		},
	}
	resp, err := v.checkIn(ctx, req)
	if err != nil {
		return nil, err
	}
	if resp.WebConnectivity == nil || len(resp.WebConnectivity.URLs) <= 0 {
		return nil, richerinput.ErrNoURLsReturned
	}
	return richerInputWrapURLs(resp.WebConnectivity.URLs), nil
}

func richerInputWrapURLs(inputs []model.OOAPIURLInfo) (outputs []richerInputTarget) {
	for _, input := range inputs {
		outputs = append(outputs, richerInputTarget{info: input})
	}
	return
}

// checkIn executes the check-in and filters the returned URLs to exclude
// the URLs that are not part of the requested categories. This is done for
// robustness, just in case we or the API do something wrong.
func (v *richerInputLoader) checkIn(
	ctx context.Context, config *model.OOAPICheckInConfig) (*model.OOAPICheckInResultNettests, error) {
	reply, err := v.sess.CheckIn(ctx, config)
	if err != nil {
		return nil, err
	}
	// Note: safe to assume that reply is not nil if err is nil
	if reply.Tests.WebConnectivity != nil && len(reply.Tests.WebConnectivity.URLs) > 0 {
		reply.Tests.WebConnectivity.URLs = richerInputPreventMistakes(
			reply.Tests.WebConnectivity.URLs, config.WebConnectivity.CategoryCodes,
		)
	}
	return &reply.Tests, nil
}

// richerInputPreventMistakes makes the code more robust with respect to any possible
// integration issue where the backend returns to us URLs that don't
// belong to the category codes we requested.
func richerInputPreventMistakes(input []model.OOAPIURLInfo, categories []string) (output []model.OOAPIURLInfo) {
	if len(categories) <= 0 {
		return input
	}
	for _, entry := range input {
		var found bool
		for _, cat := range categories {
			if entry.CategoryCode == cat {
				found = true
				break
			}
		}
		if !found {
			// TODO(bassosimone): we need to be able to print this log message again!
			//il.logger().Warnf("URL %+v not in %+v; skipping", entry, categories)
			continue
		}
		output = append(output, entry)
	}
	return
}

type richerInputMeasurer struct {
	m model.ExperimentMeasurer
}

// ExperimentName implements Measurer.
func (vm *richerInputMeasurer) ExperimentName() string {
	return vm.m.ExperimentName()
}

// ExperimentVersion implements Measurer.
func (vm *richerInputMeasurer) ExperimentVersion() string {
	return vm.m.ExperimentVersion()
}

// Run implements Measurer.
func (vm *richerInputMeasurer) Run(ctx context.Context, args *richerinput.MeasurerRunArgs[richerInputTarget]) error {
	return vm.m.Run(ctx, &model.ExperimentArgs{
		Callbacks:   args.Callbacks,
		Measurement: args.Measurement,
		Session:     args.Session,
	})
}
