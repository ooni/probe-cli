package main

import (
	"os"

	"github.com/ooni/probe-cli/v3/internal/must"
	"github.com/ooni/probe-cli/v3/internal/pipeline"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

func main() {
	rawMeasurement := must.ReadFile(os.Args[1])
	var canonicalMeasurement pipeline.CanonicalMeasurement
	must.UnmarshalJSON(rawMeasurement, &canonicalMeasurement)

	db := pipeline.NewDB()
	runtimex.Try0(db.Ingest(&canonicalMeasurement))

	must.WriteFile("db.json", must.MarshalJSON(db), 0600)

	ax := &pipeline.Analysis{}
	ax.ComputeAllValues(db)

	must.WriteFile("ax.json", must.MarshalJSON(ax), 0600)
}
