package dnsreport

import (
	"context"
	"encoding/json"
	"os"
	"sync"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// collectorWorker is the worker that writes measurement results back
func collectorWorker(
	ctx context.Context,
	wg *sync.WaitGroup,
	jsonlCacheFile string,
	outputs <-chan *Measurement,
) {
	// logging
	log.Debugf("writer for %s... started", jsonlCacheFile)
	defer log.Debugf("writer for %s... done", jsonlCacheFile)

	// synchronize with the parent
	defer wg.Done()

	// create output file
	filep := runtimex.Try1(os.Create(jsonlCacheFile))

	// write each entry
	for measurement := range outputs {
		data := runtimex.Try1(json.Marshal(measurement))
		data = append(data, '\n')
		_ = runtimex.Try1(filep.Write(data))
	}

	// make sure we flush all data
	runtimex.Try0(filep.Close())
}
