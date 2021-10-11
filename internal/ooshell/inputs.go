package ooshell

//
// inputs.go
//
// Contains code to load inputs.
//

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"math/rand"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine"
	"github.com/ooni/probe-cli/v3/internal/engine/model"
	"github.com/ooni/probe-cli/v3/internal/fsx"
)

// These errors are returned when attempting to load inputs.
var (
	ErrNoURLsReturned    = errors.New("no URLs returned")
	ErrDetectedEmptyFile = errors.New("file did not contain any input")
	ErrInputRequired     = errors.New("no input provided")
	ErrNoInputExpected   = errors.New("we did not expect any input")
)

// loadInputs loads inputs for the current experiment.
func (exp *experimentDB) loadInputs(
	ctx context.Context, builder *engine.ExperimentBuilder) ([]model.URLInfo, error) {
	inputs, err := exp.loadx(ctx, builder)
	if err != nil {
		return nil, err
	}
	if exp.env.Random {
		// TODO: should always be --random but this will break the mobile
		// app unless we do some refactoring, so for now it's optional.
		rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
		rnd.Shuffle(len(inputs), func(i, j int) {
			inputs[i], inputs[j] = inputs[j], inputs[i]
		})
	}
	return inputs, nil
}

// doLoadInputs implements loadInputs.
func (exp *experimentDB) loadx(
	ctx context.Context, builder *engine.ExperimentBuilder) ([]model.URLInfo, error) {
	switch builder.InputPolicy() {
	case engine.InputOptional:
		return exp.loadOptional()
	case engine.InputOrQueryBackend:
		return exp.loadOrQueryBackend(ctx)
	case engine.InputStrictlyRequired:
		return exp.loadStrictlyRequired(ctx)
	default:
		return exp.loadNone()
	}
}

// loadNone implements the engine.InputNone policy.
func (exp *experimentDB) loadNone() ([]model.URLInfo, error) {
	if len(exp.env.Inputs) > 0 || len(exp.env.InputFilePaths) > 0 {
		return nil, ErrNoInputExpected
	}
	// Note that we need to return a single empty entry.
	return []model.URLInfo{{}}, nil
}

// loadOptional implements the engine.InputOptional policy.
func (exp *experimentDB) loadOptional() ([]model.URLInfo, error) {
	all, err := exp.loadLocal()
	if err == nil && len(all) <= 0 {
		// Note that we need to return a single empty entry.
		all = []model.URLInfo{{}}
	}
	return all, err
}

// loadStrictlyRequired implements the engine.InputStrictlyRequired policy.
func (exp *experimentDB) loadStrictlyRequired(ctx context.Context) ([]model.URLInfo, error) {
	all, err := exp.loadLocal()
	if err != nil || len(all) > 0 {
		return all, err
	}
	return nil, ErrInputRequired
}

// loadOrQueryBackend implements the engine.InputOrQueryBackend policy.
func (exp *experimentDB) loadOrQueryBackend(ctx context.Context) ([]model.URLInfo, error) {
	all, err := exp.loadLocal()
	if err != nil || len(all) > 0 {
		return all, err
	}
	return exp.loadRemote(ctx)
}

// loadLocal loads inputs from env.Inputs and env.InputFilePaths.
func (exp *experimentDB) loadLocal() ([]model.URLInfo, error) {
	inputs := []model.URLInfo{}
	for _, input := range exp.env.Inputs {
		inputs = append(inputs, model.URLInfo{URL: input})
	}
	for _, filepath := range exp.env.InputFilePaths {
		extra, err := exp.readfile(filepath, fsx.OpenFile)
		if err != nil {
			return nil, err
		}
		// See https://github.com/ooni/probe-engine/issues/1123.
		if len(extra) <= 0 {
			return nil, fmt.Errorf("%w: %s", ErrDetectedEmptyFile, filepath)
		}
		inputs = append(inputs, extra...)
	}
	return inputs, nil
}

// readfile reads inputs from the specified file. The open argument should be
// compatible with stdlib's fs.Open and helps us with unit testing.
func (exp *experimentDB) readfile(filepath string,
	openFunc func(filepath string) (fs.File, error)) ([]model.URLInfo, error) {
	inputs := []model.URLInfo{}
	filep, err := openFunc(filepath)
	if err != nil {
		return nil, err
	}
	defer filep.Close()
	// Implementation note: when you save file with vim, you have newline at
	// end of file and you don't want to consider that an input line. While there
	// ignore any other empty line that may occur inside the file.
	scanner := bufio.NewScanner(filep)
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			inputs = append(inputs, model.URLInfo{URL: line})
		}
	}
	if scanner.Err() != nil {
		return nil, scanner.Err()
	}
	return inputs, nil
}

// loadRemote loads inputs from a remote source.
func (exp *experimentDB) loadRemote(ctx context.Context) ([]model.URLInfo, error) {
	config := &model.CheckInConfig{
		Charging:        exp.env.Charging,
		OnWiFi:          exp.env.OnWiFi,
		Platform:        exp.sess.Platform(),
		ProbeASN:        exp.sess.ProbeASNString(),
		ProbeCC:         exp.sess.ProbeCC(),
		RunType:         exp.env.RunType,
		SoftwareName:    exp.sess.SoftwareName(),
		SoftwareVersion: exp.sess.SoftwareVersion(),
		WebConnectivity: model.CheckInConfigWebConnectivity{
			CategoryCodes: exp.env.Categories,
		},
	}
	reply, err := exp.checkIn(ctx, config)
	if err != nil {
		return nil, err
	}
	if reply.WebConnectivity == nil || len(reply.WebConnectivity.URLs) <= 0 {
		return nil, ErrNoURLsReturned
	}
	return reply.WebConnectivity.URLs, nil
}

// checkIn executes the check-in and filters the returned URLs to exclude
// the URLs that are not part of the requested categories. This is done for
// robustness, just in case we or the API do something wrong.
func (exp *experimentDB) checkIn(
	ctx context.Context, config *model.CheckInConfig) (*model.CheckInInfo, error) {
	reply, err := exp.sess.CheckIn(ctx, config)
	if err != nil {
		return nil, err
	}
	// Note: safe to assume that reply is not nil if err is nil
	if reply.WebConnectivity != nil && len(reply.WebConnectivity.URLs) > 0 {
		reply.WebConnectivity.URLs = exp.preventMistakes(
			reply.WebConnectivity.URLs,
			config.WebConnectivity.CategoryCodes,
		)
	}
	return reply, nil
}

// preventMistakes makes the code more robust with respect to any possible
// integration issue where the backend returns to us URLs that don't
// belong to the category codes we requested.
func (exp *experimentDB) preventMistakes(
	input []model.URLInfo, categories []string) (output []model.URLInfo) {
	if len(categories) <= 0 {
		return input
	}
	cats := make(map[string]bool)
	for _, cat := range categories {
		cats[cat] = true
	}
	for _, entry := range input {
		if _, found := cats[entry.CategoryCode]; !found {
			exp.logger.Warnf("URL %+v not in %+v; skipping", entry, categories)
			continue
		}
		output = append(output, entry)
	}
	return
}
