package nettests

import (
	"bufio"
	"errors"
	"fmt"

	"github.com/ooni/probe-cli/v3/internal/fsx"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// ErrDetectedEmptyFile indicates we detected an open file.
var ErrDetectedEmptyFile = errors.New("inputloader: file did not contain any input")

// ErrNoAvailableInput indicates there's no available input.
var ErrNoAvailableInput = errors.New("inputloader: no available input")

// loadInputs loads inputs by giving priority to user-defined
// input and falling back to API-provided inputs.
func loadInputs(
	apiInputs []model.OOAPIURLInfo,
	userFiles []string,
	userInputs []string,
) ([]model.OOAPIURLInfo, error) {
	inputs, err := loadLocalInputs(userFiles, userInputs)
	if err != nil || len(inputs) > 0 {
		return inputs, err
	}
	if len(apiInputs) <= 0 {
		return nil, ErrNoAvailableInput
	}
	return apiInputs, nil
}

// loadLocalInputs loads inputs from user-provided inputs and files.
func loadLocalInputs(userFiles, userInputs []string) ([]model.OOAPIURLInfo, error) {
	inputs := []model.OOAPIURLInfo{}
	for _, input := range userInputs {
		inputs = append(inputs, model.OOAPIURLInfo{URL: input})
	}
	for _, filepath := range userFiles {
		extra, err := loadLocalInputFile(filepath)
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

// loadLocalInputFile reads inputs from the specified file.
func loadLocalInputFile(filepath string) ([]model.OOAPIURLInfo, error) {
	inputs := []model.OOAPIURLInfo{}
	filep, err := fsx.OpenFile(filepath)
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
			inputs = append(inputs, model.OOAPIURLInfo{URL: line})
		}
	}
	if scanner.Err() != nil {
		return nil, scanner.Err()
	}
	return inputs, nil
}
