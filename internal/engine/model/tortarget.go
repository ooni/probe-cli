package model

// TorTarget is a target for the tor experiment.
type TorTarget struct {
	// Address is the address of the target.
	Address string `json:"address"`

	// Name is the name of the target.
	Name string `json:"name"`

	// Params contains optional params for, e.g., pluggable transports.
	Params map[string][]string `json:"params"`

	// Protocol is the protocol to use with the target.
	Protocol string `json:"protocol"`

	// Source is the source from which we fetched this specific
	// target. Whenever the source is non-empty, we will treat
	// this specific target as a private target.
	Source string `json:"source"`
}
