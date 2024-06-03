package richerinput

//
// Definition of InputLoader and its implementations
//

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/ooni/probe-cli/v3/internal/fsx"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// ErrNoURLsReturned indicates that a OONI backend API did not return any URL.
var ErrNoURLsReturned = errors.New("no URLs returned")

// InputLoader loads richer input.
type InputLoader[T Target] interface {
	Load(ctx context.Context, config *model.RicherInputConfig) ([]T, error)
}

// VoidInputLoader is a the loader to use when there's no richer input.
type VoidInputLoader struct{}

var _ InputLoader[VoidTarget] = &VoidInputLoader{}

// ErrNoInputExpected is the error returned when we did not expect any input.
var ErrNoInputExpected = errors.New("we did not expect any input")

// Load implements InputLoader.
func (v *VoidInputLoader) Load(ctx context.Context, config *model.RicherInputConfig) ([]VoidTarget, error) {
	if config.ContainsUserConfiguredInput() {
		return nil, ErrNoInputExpected
	}
	return []VoidTarget{{}}, nil
}

// ErrDetectedEmptyFile indicates that we attempted to read inputs from an empty file.
var ErrDetectedEmptyFile = errors.New("file did not contain any input")

// LoadInputs loads inputs from the given [model.RicherInputConfig].
func LoadInputs(config *model.RicherInputConfig) ([]string, error) {
	// Start by loading inputs from the static inputs.
	inputs := append([]string{}, config.Inputs...)

	// Then load inputs from each of the given file paths.
	for _, filepath := range config.InputFilePaths {

		// Read from one of the files.
		extra, err := inputLoaderReadFile(filepath, fsx.OpenFile)

		// Handle the case of I/O error.
		if err != nil {
			return nil, err
		}

		// Handle the case of empty file.
		//
		// See https://github.com/ooni/probe-engine/issues/1123.
		if len(extra) <= 0 {
			return nil, fmt.Errorf("%w: %s", ErrDetectedEmptyFile, filepath)
		}

		// Append the results to the current inputs.
		inputs = append(inputs, extra...)
	}

	return inputs, nil
}

// inputLoaderOpenFn is the type of the function to open a file.
type inputLoaderOpenFn func(filepath string) (fs.File, error)

// inputLoaderReadFilereadfile reads inputs from the specified file. The open argument
// should be compatible with stdlib's fs.Open and helps us with unit testing.
func inputLoaderReadFile(filepath string, open inputLoaderOpenFn) ([]string, error) {
	inputs := []string{}

	// Try to open the given file path.
	filep, err := open(filepath)

	// Handle the case where we can't open it.
	if err != nil {
		return nil, err
	}

	// Make sure we don't leak the fileptr.
	defer filep.Close()

	// Read each line of the file.
	//
	// Implementation note: when you save file with vim, you have newline at
	// end of file and you don't want to consider that an input line. While there
	// ignore any other empty line that may occur inside the file.
	scanner := bufio.NewScanner(filep)
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			inputs = append(inputs, line)
		}
	}

	// Handle errors occurred while reading the file.
	if scanner.Err() != nil {
		return nil, scanner.Err()
	}

	return inputs, nil
}
