package oonirun

//
// OONI Run v2 implementation
//

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/hexops/gotextdiff"
	"github.com/hexops/gotextdiff/myers"
	"github.com/hexops/gotextdiff/span"
	"github.com/ooni/probe-cli/v3/internal/kvstore"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// v2Descriptor describes a single nettest to run.
type v2Descriptor struct {
	// NettestArguments contains the arguments for the nettest.
	NettestArguments v2Arguments `json:"ta"`

	// NettestName is the name of the nettest to run.
	NettestName string `json:"tn"`
}

// v2Arguments contains arguments for a given nettest.
type v2Arguments struct {
	// Inputs contains inputs for the experiment.
	Inputs []string `json:"inputs"`

	// Options contains the experiment options.
	Options map[string]any `json:"options"`
}

// ErrHTTPRequestFailed indicates that an HTTP request failed.
var ErrHTTPRequestFailed = errors.New("oonirun: HTTP request failed")

// getV2DescriptorsFromHTTPSURL GETs a list of v2Descriptor from
// a static URL (e.g., from a GitHub repo or from a Gist).
func getV2DescriptorsFromHTTPSURL(
	ctx context.Context, client model.HTTPClient, URL string) ([]v2Descriptor, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", URL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, ErrHTTPRequestFailed
	}
	reader := io.LimitReader(resp.Body, 1<<22)
	data, err := netxlite.ReadAllContext(ctx, reader)
	if err != nil {
		return nil, err
	}
	var descs []v2Descriptor
	if err := json.Unmarshal(data, &descs); err != nil {
		return nil, err
	}
	return descs, nil
}

// v2DescriptorCache contains all the known v2Descriptor entries.
type v2DescriptorCache struct {
	// Entries contains all the cached descriptors.
	Entries map[string][]v2Descriptor
}

// v2DescriptorCacheKey is the name of the kvstore2 entry keeping
// information about already known v2Descripor instances.
const v2DescriptorCacheKey = "oonirun-v2.state"

// v2DescriptorCacheLoad loads the v2DescriptorCache.
func v2DescriptorCacheLoad(fsstore model.KeyValueStore) (*v2DescriptorCache, error) {
	data, err := fsstore.Get(v2DescriptorCacheKey)
	if err != nil {
		if errors.Is(err, kvstore.ErrNoSuchKey) {
			cache := &v2DescriptorCache{
				Entries: make(map[string][]v2Descriptor),
			}
			return cache, nil
		}
		return nil, err
	}
	var cache v2DescriptorCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, err
	}
	if cache.Entries == nil {
		cache.Entries = make(map[string][]v2Descriptor)
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
// - oldValue is the old v2Descriptor, which may be empty;
//
// - newValue is the new v2Descriptor;
//
// - err is the error that occurred, or nil in case of success.
func (cache *v2DescriptorCache) PullChangesWithoutSideEffects(ctx context.Context,
	client model.HTTPClient, URL string) (oldValue, newValue []v2Descriptor, err error) {
	oldValue = cache.Entries[URL]
	newValue, err = getV2DescriptorsFromHTTPSURL(ctx, client, URL)
	return
}

// Update updates the given cache entry and writes back onto the disk.
func (cache *v2DescriptorCache) Update(
	fsstore model.KeyValueStore, URL string, entry []v2Descriptor) error {
	// Note: NOT SAFE for concurrent use (default for methods)
	cache.Entries[URL] = entry
	data, err := json.Marshal(cache)
	if err != nil {
		return err
	}
	return fsstore.Set(v2DescriptorCacheKey, data)
}

// v2MeasureDescriptors performs the measurement or measurements
// described by the given list of v2Descriptor.
func v2MeasureDescriptors(ctx context.Context, config *Config, descs []v2Descriptor) error {
	logger := config.Session.Logger()
	for _, desc := range descs {
		if desc.NettestName == "" {
			logger.Warn("nettest name cannot be empty")
			continue
		}
		exp := &Experiment{
			Annotations:    config.Annotations,
			ExtraOptions:   desc.NettestArguments.Options,
			Inputs:         desc.NettestArguments.Inputs,
			InputFilePaths: nil,
			MaxRuntime:     config.MaxRuntime,
			Name:           desc.NettestName,
			NoCollector:    config.NoCollector,
			NoJSON:         config.NoJSON,
			Random:         config.Random,
			ReportFile:     config.ReportFile,
			Session:        config.Session,
		}
		if err := exp.Run(ctx); err != nil {
			logger.Warnf("cannot run experiment: %s", err.Error())
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
func v2DescriptorDiff(oldValue, newValue []v2Descriptor, URL string) string {
	oldData, err := json.MarshalIndent(oldValue, "", "  ")
	runtimex.PanicOnError(err, "json.MarshalIndent failed unexpectedly")
	newData, err := json.MarshalIndent(newValue, "", "  ")
	runtimex.PanicOnError(err, "json.MarshalIndent failed unexpectedly")
	oldString, newString := string(oldData)+"\n", string(newData)+"\n"
	oldFile := "OLD " + URL
	newFile := "NEW " + URL
	edits := myers.ComputeEdits(span.URIFromPath(oldFile), oldString, newString)
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
func v2MeasureHTTPS(ctx context.Context, config *Config, URL string) error {
	cache, err := v2DescriptorCacheLoad(config.KVStore)
	if err != nil {
		return err
	}
	clnt := config.Session.DefaultHTTPClient()
	oldValue, newValue, err := cache.PullChangesWithoutSideEffects(ctx, clnt, URL)
	if err != nil {
		return err
	}
	diff := v2DescriptorDiff(oldValue, newValue, URL)
	if !config.AcceptChanges && diff != "" {
		logger := config.Session.Logger()
		logger.Warnf("oonirun: %s changed as follows:\n\n%s", URL, diff)
		logger.Warnf("oonirun: we are not going to run this link until you accept changes")
		return ErrNeedToAcceptChanges
	}
	if diff != "" {
		if err := cache.Update(config.KVStore, URL, newValue); err != nil {
			return err
		}
	}
	return v2MeasureDescriptors(ctx, config, newValue)
}
