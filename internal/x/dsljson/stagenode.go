package dsljson

import "encoding/json"

// StageNode describes a stage node in the DSL.
type StageNode struct {
	Name  string          `json:"name"`
	Value json.RawMessage `json:"value"`
}
