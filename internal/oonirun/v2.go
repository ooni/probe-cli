package oonirun

//
// OONI Run v2 implementation
//

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/hexops/gotextdiff"
	"github.com/hexops/gotextdiff/myers"
	"github.com/hexops/gotextdiff/span"
	"github.com/ooni/probe-cli/v3/internal/httpclientx"
	"github.com/ooni/probe-cli/v3/internal/kvstore"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

var (
	// v2CountEmptyNettestNames counts the number of cases in which we have been
	// given an empty nettest name, which is useful for testing.
	v2CountEmptyNettestNames = &atomic.Int64{}

	// v2CountFailedExperiments countes the number of failed experiments
	// and is useful when testing this package
	v2CountFailedExperiments = &atomic.Int64{}
)

// V2Descriptor describes a list of nettests to run together.
type V2Descriptor struct {
	// Name is the name of this descriptor.
	Name string `json:"name"`

	// Description contains a long description.
	Description string `json:"description"`

	// Author contains the author's name.
	Author string `json:"author"`

	// Nettests contains the list of nettests to run.
	Nettests []V2Nettest `json:"nettests"`
}

// V2Nettest specifies how a nettest should run.
type V2Nettest struct {
	// Inputs contains inputs for the experiment.
	Inputs []string `json:"inputs"`

	// Options contains the experiment options. Any option name starting with
	// `Safe` will be available for the experiment run, but omitted from
	// the serialized Measurement that the experiment builder will submit
	// to the OONI backend.
	Options map[string]any `json:"options"`

	// TestName contains the nettest name.
	TestName string `json:"test_name"`
}

// getV2DescriptorFromHTTPSURL GETs a v2Descriptor instance from
// a static URL (e.g., from a GitHub repo or from a Gist).
func getV2DescriptorFromHTTPSURL(ctx context.Context, client model.HTTPClient,
	logger model.Logger, URL string) (*V2Descriptor, error) {
	return httpclientx.GetJSON[*V2Descriptor](ctx, URL, &httpclientx.Config{
		Authorization: "", // not needed
		Client:        client,
		Logger:        logger,
		UserAgent:     model.HTTPHeaderUserAgent,
	})
}

// v2DescriptorCache contains all the known v2Descriptor entries.
type v2DescriptorCache struct {
	// Entries contains all the cached descriptors.
	Entries map[string]*V2Descriptor
}

// v2DescriptorCacheKey is the name of the kvstore2 entry keeping
// information about already known v2Descriptor instances.
const v2DescriptorCacheKey = "oonirun-v2.state"

// v2DescriptorCacheLoad loads the v2DescriptorCache.
func v2DescriptorCacheLoad(fsstore model.KeyValueStore) (*v2DescriptorCache, error) {
	// attempt to access the cache
	data, err := fsstore.Get(v2DescriptorCacheKey)

	// if there's a miss either create a new descriptor or return the
	// error if it's something I/O related
	if err != nil {
		if errors.Is(err, kvstore.ErrNoSuchKey) {
			cache := &v2DescriptorCache{
				Entries: make(map[string]*V2Descriptor),
			}
			return cache, nil
		}
		return nil, err
	}

	// transform the raw descriptor into a struct
	var cache v2DescriptorCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, err
	}

	// handle the case where there are no entries inside the on-disk cache
	// by properly initializing to a non-nil map
	if cache.Entries == nil {
		cache.Entries = make(map[string]*V2Descriptor)
	}

	return &cache, nil
}

// PullChangesWithoutSideEffects fetches v2Descriptor changes.
//
// This function DOES NOT change the state of the cache. It just returns to
// the caller what changed for a given entry. It is up-to-the-caller to choose
// what to do in case there are changes depending on the CLI flags.
//
// Arguments:
//
// - ctx is the context for deadline/cancellation;
//
// - client is the HTTPClient to use;
//
// - URL is the URL from which to download/update the OONIRun v2Descriptor.
//
// Return values:
//
// - oldValue is the old v2Descriptor, which may be nil;
//
// - newValue is the new v2Descriptor, which may be nil;
//
// - err is the error that occurred, or nil in case of success.
func (cache *v2DescriptorCache) PullChangesWithoutSideEffects(
	ctx context.Context, client model.HTTPClient, logger model.Logger,
	URL string) (oldValue, newValue *V2Descriptor, err error) {
	oldValue = cache.Entries[URL]
	newValue, err = getV2DescriptorFromHTTPSURL(ctx, client, logger, URL)
	return
}

// Update updates the given cache entry and writes back onto the disk.
//
// Note: this method modifies cache and is not safe for concurrent usage.
func (cache *v2DescriptorCache) Update(
	fsstore model.KeyValueStore, URL string, entry *V2Descriptor) error {
	cache.Entries[URL] = entry
	data, err := json.Marshal(cache)
	runtimex.PanicOnError(err, "json.Marshal failed")
	return fsstore.Set(v2DescriptorCacheKey, data)
}

// ErrNilDescriptor indicates that we have been passed a descriptor that is nil.
var ErrNilDescriptor = errors.New("oonirun: descriptor is nil")

// V2MeasureDescriptor performs the measurement or measurements
// described by the given list of v2Descriptor.
func V2MeasureDescriptor(ctx context.Context, config *LinkConfig, desc *V2Descriptor) error {
	if desc == nil {
		// Note: we have a test checking that we can handle a nil
		// descriptor, yet adding also this extra safety net feels
		// more robust in terms of the implementation.
		return ErrNilDescriptor
	}

	logger := config.Session.Logger()

	for _, nettest := range desc.Nettests {
		// early handling of the case where the test name is empty
		if nettest.TestName == "" {
			logger.Warn("oonirun: nettest name cannot be empty")
			v2CountEmptyNettestNames.Add(1)
			continue
		}

		// construct an experiment from the current nettest
		exp := &Experiment{
			Annotations:            config.Annotations,
			ExtraOptions:           nettest.Options,
			Inputs:                 nettest.Inputs,
			InputFilePaths:         nil,
			MaxRuntime:             config.MaxRuntime,
			Name:                   nettest.TestName,
			NoCollector:            config.NoCollector,
			NoJSON:                 config.NoJSON,
			Random:                 config.Random,
			ReportFile:             config.ReportFile,
			Session:                config.Session,
			newExperimentBuilderFn: nil,
			newInputLoaderFn:       nil,
			newSubmitterFn:         nil,
			newSaverFn:             nil,
			newInputProcessorFn:    nil,
		}

		// actually run the experiment
		if err := exp.Run(ctx); err != nil {
			logger.Warnf("cannot run experiment: %s", err.Error())
			v2CountFailedExperiments.Add(1)
			continue
		}
	}

	return nil
}

// ErrNeedToAcceptChanges indicates that the user needs to accept
// changes (i.e., a new or modified set of descriptors) before
// we can actually run this set of descriptors.
var ErrNeedToAcceptChanges = errors.New("oonirun: need to accept changes")

// v2DescriptorDiff shows what changed between the old and the new descriptors.
func v2DescriptorDiff(oldValue, newValue *V2Descriptor, URL string) string {
	// JSON serialize old descriptor
	oldData, err := json.MarshalIndent(oldValue, "", "  ")
	runtimex.PanicOnError(err, "json.MarshalIndent failed unexpectedly")

	// JSON serialize new descriptor
	newData, err := json.MarshalIndent(newValue, "", "  ")
	runtimex.PanicOnError(err, "json.MarshalIndent failed unexpectedly")

	// make sure the serializations are newline-terminated
	oldString, newString := string(oldData)+"\n", string(newData)+"\n"

	// generate names for the final diff
	oldFile := "OLD " + URL
	newFile := "NEW " + URL

	// compute the edits to update from the old to the new descriptor
	edits := myers.ComputeEdits(span.URIFromPath(oldFile), oldString, newString)

	// transform the edits and obtain an unified diff
	return fmt.Sprint(gotextdiff.ToUnified(oldFile, newFile, oldString, edits))
}

// v2MeasureHTTPS performs a measurement using an HTTPS v2 OONI Run URL
// and returns whether performing this measurement failed.
//
// This function maintains an on-disk cache that tracks the status of
// OONI Run v2 links. If there are any changes and the user has not
// provided config.AcceptChanges, this function will log what has changed
// and will return with an ErrNeedToAcceptChanges error.
//
// In such a case, the caller SHOULD print additional information
// explaining how to accept changes and then SHOULD exit 1 or similar.
func v2MeasureHTTPS(ctx context.Context, config *LinkConfig, URL string) error {
	logger := config.Session.Logger()
	logger.Infof("oonirun/v2: running %s", URL)

	// load the descriptor from the cache
	cache, err := v2DescriptorCacheLoad(config.KVStore)
	if err != nil {
		return err
	}

	// pull a possibly new descriptor without updating the old descriptor
	clnt := config.Session.DefaultHTTPClient()
	oldValue, newValue, err := cache.PullChangesWithoutSideEffects(ctx, clnt, logger, URL)
	if err != nil {
		return err
	}

	// compare the new descriptor to the old descriptor
	diff := v2DescriptorDiff(oldValue, newValue, URL)

	// possibly stop if configured to ask for permission when accepting changes
	if !config.AcceptChanges && diff != "" {
		logger.Warnf("oonirun: %s changed as follows:\n\n%s", URL, diff)
		logger.Warnf("oonirun: we are not going to run this link until you accept changes")
		return ErrNeedToAcceptChanges
	}

	// in case there are changes, update the descriptor
	if diff != "" {
		if err := cache.Update(config.KVStore, URL, newValue); err != nil {
			return err
		}
	}

	// measure using the possibly-new descriptor
	//
	// note: this function gracefully handles nil values
	return V2MeasureDescriptor(ctx, config, newValue)
}
