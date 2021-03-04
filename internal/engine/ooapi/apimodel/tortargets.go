package apimodel

// TorTargetsRequest is a request for the TorTargets API.
type TorTargetsRequest struct{}

// TorTargetsResponse is the response from the TorTargets API.
type TorTargetsResponse map[string]TorTargetsTarget

// TorTargetsTarget is a target for the tor experiment.
type TorTargetsTarget struct {
	Address  string              `json:"address"`
	Name     string              `json:"name"`
	Params   map[string][]string `json:"params"`
	Protocol string              `json:"protocol"`
	Source   string              `json:"source"`
}
