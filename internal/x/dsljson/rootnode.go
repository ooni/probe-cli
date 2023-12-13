package dsljson

// RootNode is the root node of the DSL.
type RootNode struct {
	Stages []StageNode `json:"stages"`
}
