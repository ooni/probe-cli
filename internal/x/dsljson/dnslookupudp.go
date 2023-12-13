package dsljson

import (
	"encoding/json"

	"github.com/ooni/probe-cli/v3/internal/x/dslvm"
)

type dnsLookupUDPValue struct {
	Domain   string   `json:"domain"`
	Output   string   `json:"output"`
	Resolver string   `json:"resolver"`
	Tags     []string `json:"tags"`
}

func (lx *loader) onDNSLookupUDP(raw json.RawMessage) error {
	// parse the raw value
	var value dnsLookupUDPValue
	if err := json.Unmarshal(raw, &value); err != nil {
		return err
	}

	// create the required output registers
	output, err := registerMakeOutput[string](lx, value.Output)
	if err != nil {
		return err
	}

	// instantiate the ASM stage
	sx := &dslvm.DNSLookupUDPStage{
		Domain:   value.Domain,
		Output:   output,
		Resolver: value.Resolver,
		Tags:     value.Tags,
	}

	// remember the stage for later
	lx.stages = append(lx.stages, sx)
	return nil
}
