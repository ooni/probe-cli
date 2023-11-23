package pipeline

import (
	"fmt"
	"path/filepath"

	"github.com/ooni/probe-cli/v3/internal/must"
	"github.com/ooni/probe-cli/v3/internal/optional"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

type flatDB struct {
	XdnsByTxID    map[int64]*DNSObservation
	XthEpntByEpnt map[string]*EndpointObservationTH
	XthWeb        optional.Value[*WebObservationTH]
	XwebByTxID    map[int64]*WebEndpointObservation
}

func Example() {
	rawJSON := must.ReadFile(filepath.Join("testdata", "youtube.json"))
	var meas CanonicalMeasurement
	must.UnmarshalJSON(rawJSON, &meas)

	db := NewDB()
	runtimex.Try0(db.Ingest(&meas))

	fdb := &flatDB{
		XdnsByTxID:    db.dnsByTxID,
		XthEpntByEpnt: db.thEpntByEpnt,
		XthWeb:        db.thWeb,
		XwebByTxID:    db.webByTxID,
	}
	rawDB := must.MarshalJSON(fdb)
	must.WriteFile(filepath.Join("testdata", "youtube_db.json"), rawDB, 0600)

	fmt.Printf("true\n")
	// Output:
	// true
}
