package pipeline

import (
	"fmt"
	"path/filepath"

	"github.com/ooni/probe-cli/v3/internal/must"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

func Example() {
	rawJSON := must.ReadFile(filepath.Join("testdata", "youtube.json"))
	var meas CanonicalMeasurement
	must.UnmarshalJSON(rawJSON, &meas)

	db := NewDB()
	runtimex.Try0(db.Ingest(&meas))

	rawDB := must.MarshalJSON(db)
	must.WriteFile(filepath.Join("testdata", "youtube_db.json"), rawDB, 0600)

	ax := &Analysis{}
	ax.ComputeAllValues(db)

	rawAx := must.MarshalJSON(ax)
	must.WriteFile(filepath.Join("testdata", "youtube_ax.json"), rawAx, 0600)

	fmt.Printf("true\n")
	// Output:
	// true
}
