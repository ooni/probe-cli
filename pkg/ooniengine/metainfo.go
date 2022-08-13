package main

//
// Meta info tasks
//

import (
	"github.com/ooni/probe-cli/v3/internal/engine"
	"github.com/ooni/probe-cli/v3/pkg/ooniengine/abi"
)

// experimentMetaInfoCall implements OONICall for ExperimentMetaInfo.
func experimentMetaInfoCall(args []byte) (out *goMessage) {
	value := &abi.ExperimentMetaInfoResponse{}
	out = &goMessage{
		key:   "ExperimentMetaInfoResponse",
		value: value,
	}
	for _, exp := range engine.AllExperimentsInfo() {
		entry := &abi.ExperimentMetaInfoEntry{
			Name:      exp.Name,
			UsesInput: exp.InputPolicy != engine.InputNone,
		}
		value.Entry = append(value.Entry, entry)
	}
	return out
}
