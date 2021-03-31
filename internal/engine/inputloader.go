package engine

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/ooni/probe-cli/v3/internal/engine/internal/fsx"
	"github.com/ooni/probe-cli/v3/internal/engine/model"
)

// These errors are returned by the InputLoader.
var (
	ErrNoURLsReturned    = errors.New("no URLs returned")
	ErrDetectedEmptyFile = errors.New("file did not contain any input")
	ErrInputRequired     = errors.New("no input provided")
	ErrNoInputExpected   = errors.New("we did not expect any input")
)

// InputLoaderSession is the session according to an InputLoader. We
// introduce this abstraction because it helps us with testing.
type InputLoaderSession interface {
	CheckIn(ctx context.Context,
		config *model.CheckInConfig) (*model.CheckInInfo, error)
}

// InputLoader loads input according to the specified policy
// either from command line and input files or from OONI services. The
// behaviour depends on the input policy as described below.
//
// You MUST NOT change any public field of this structure when
// in use, because that MAY lead to data races.
//
// InputNone
//
// We fail if there is any StaticInput or any SourceFiles. If
// there's no input, we return a single, empty entry that causes
// experiments that don't require input to run once.
//
// InputOptional
//
// We gather input from StaticInput and SourceFiles. If there is
// input, we return it. Otherwise we return a single, empty entry
// that causes experiments that don't require input to run once.
//
// InputOrQueryBackend
//
// We gather input from StaticInput and SourceFiles. If there is
// input, we return it. Otherwise, we use OONI's probe services
// to gather input using the best API for the task.
//
// InputStrictlyRequired
//
// We gather input from StaticInput and SourceFiles. If there is
// input, we return it. Otherwise, we return an error.
type InputLoader struct {
	// CheckInConfig contains options for the CheckIn API. If
	// not set, then we'll create a default config. If set but
	// there are fields inside it that are not set, then we
	// will set them to a default value.
	CheckInConfig *model.CheckInConfig

	// InputPolicy specifies the input policy for the
	// current experiment. We will not load any input if
	// the policy says we should not. You MUST fill in
	// this field.
	InputPolicy InputPolicy

	// Session is the current measurement session. You
	// MUST fill in this field.
	Session InputLoaderSession

	// StaticInputs contains optional input to be added
	// to the resulting input list if possible.
	StaticInputs []string

	// SourceFiles contains optional files to read input
	// from. Each file should contain a single input string
	// per line. We will fail if any file is unreadable
	// as well as if any file is empty.
	SourceFiles []string
}

// Load attempts to load input using the specified input loader. We will
// return a list of URLs because this is the only input we support.
func (il *InputLoader) Load(ctx context.Context) ([]model.URLInfo, error) {
	switch il.InputPolicy {
	case InputOptional:
		return il.loadOptional()
	case InputOrQueryBackend:
		return il.loadOrQueryBackend(ctx)
	case InputStrictlyRequired:
		return il.loadStrictlyRequired(ctx)
	default:
		return il.loadNone()
	}
}

// loadNone implements the InputNone policy.
func (il *InputLoader) loadNone() ([]model.URLInfo, error) {
	if len(il.StaticInputs) > 0 || len(il.SourceFiles) > 0 {
		return nil, ErrNoInputExpected
	}
	// Note that we need to return a single empty entry.
	return []model.URLInfo{{}}, nil
}

// loadOptional implements the InputOptional policy.
func (il *InputLoader) loadOptional() ([]model.URLInfo, error) {
	inputs, err := il.loadLocal()
	if err == nil && len(inputs) <= 0 {
		// Note that we need to return a single empty entry.
		inputs = []model.URLInfo{{}}
	}
	return inputs, err
}

// loadStrictlyRequired implements the InputStrictlyRequired policy.
func (il *InputLoader) loadStrictlyRequired(ctx context.Context) ([]model.URLInfo, error) {
	inputs, err := il.loadLocal()
	if err != nil || len(inputs) > 0 {
		return inputs, err
	}
	return nil, ErrInputRequired
}

// loadOrQueryBackend implements the InputOrQueryBackend policy.
func (il *InputLoader) loadOrQueryBackend(ctx context.Context) ([]model.URLInfo, error) {
	inputs, err := il.loadLocal()
	if err != nil || len(inputs) > 0 {
		return inputs, err
	}
	return il.loadRemote(ctx)
}

// loadLocal loads inputs from StaticInputs and SourceFiles.
func (il *InputLoader) loadLocal() ([]model.URLInfo, error) {
	inputs := []model.URLInfo{}
	for _, input := range il.StaticInputs {
		inputs = append(inputs, model.URLInfo{URL: input})
	}
	for _, filepath := range il.SourceFiles {
		extra, err := il.readfile(filepath, fsx.Open)
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

// inputLoaderOpenFn is the type of the function to open a file.
type inputLoaderOpenFn func(filepath string) (fs.File, error)

// readfile reads inputs from the specified file. The open argument should be
// compatible with stdlib's fs.Open and helps us with unit testing.
func (il *InputLoader) readfile(filepath string, open inputLoaderOpenFn) ([]model.URLInfo, error) {
	inputs := []model.URLInfo{}
	filep, err := open(filepath)
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
func (il *InputLoader) loadRemote(ctx context.Context) ([]model.URLInfo, error) {
	config := il.CheckInConfig
	if config == nil {
		// Note: Session.CheckIn documentation says it will fill in
		// any field with a required value with a reasonable default
		// if such value is missing. So, here we just need to be
		// concerned about NOT passing it a NULL pointer.
		config = &model.CheckInConfig{}
	}
	reply, err := il.Session.CheckIn(ctx, config)
	if err != nil {
		return nil, err
	}
	if reply.WebConnectivity == nil || len(reply.WebConnectivity.URLs) <= 0 {
		return nil, ErrNoURLsReturned
	}
	return reply.WebConnectivity.URLs, nil
}
