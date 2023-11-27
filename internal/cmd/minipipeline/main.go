package main

import (
	"fmt"
	"os"

	"github.com/ooni/probe-cli/v3/internal/minipipeline"
	"github.com/ooni/probe-cli/v3/internal/must"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

var (
	mustWriteFileFn = must.WriteFile
	inputs          = os.Args[1:]
)

func main() {
	for idx, name := range inputs {
		var meas minipipeline.Measurement
		must.UnmarshalJSON(must.ReadFile(name), &meas)
		container := runtimex.Try1(minipipeline.LoadWebMeasurement(&meas))
		mustWriteFileFn(fmt.Sprintf("observations-%010d.json", idx), must.MarshalJSON(container), 0600)
		analysis := minipipeline.AnalyzeWebMeasurement(container)
		mustWriteFileFn(fmt.Sprintf("analysis-%010d.json", idx), must.MarshalJSON(analysis), 0600)
	}
}
