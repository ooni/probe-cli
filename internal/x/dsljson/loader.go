package dsljson

import (
	"encoding/json"
	"fmt"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/must"
	"github.com/ooni/probe-cli/v3/internal/x/dslvm"
)

type loader struct {
	// gone contains the names of registers we have already used.
	gone map[string]bool

	// loaders contains the loaders.
	loaders map[string]func(json.RawMessage) error

	// registers maps variable names to values.
	registers map[string]any

	// stages contains the stages of DSL ASM stages.
	stages []dslvm.Stage

	// toWait contains the channels to wait for.
	toWait []<-chan dslvm.Done
}

func newLoader() *loader {
	lx := &loader{
		gone:      map[string]bool{},
		loaders:   make(map[string]func(json.RawMessage) error),
		registers: map[string]any{},
		stages:    []dslvm.Stage{},
	}

	lx.loaders["drop"] = lx.onDrop
	lx.loaders["dns_lookup_udp"] = lx.onDNSLookupUDP
	lx.loaders["dedup_addrs"] = lx.onDedupAddrs
	lx.loaders["getaddrinfo"] = lx.onGetaddrinfo
	lx.loaders["http_round_trip"] = lx.onHTTPRoundTrip
	lx.loaders["make_endpoints"] = lx.onMakeEndpoints
	lx.loaders["quic_handshake"] = lx.onQUICHandshake
	lx.loaders["take_n"] = lx.onTakeN
	lx.loaders["tcp_connect"] = lx.onTCPConnect
	lx.loaders["tls_handshake"] = lx.onTLSHandshake
	lx.loaders["tee_addrs"] = lx.onTeeAddrs

	return lx
}

func (lx *loader) load(logger model.Logger, root *RootNode) error {

	// load all the stages that belong to the root node
	if err := lx.loadStages(logger, root.Stages...); err != nil {
		return err
	}

	// insert missing drops inside the code
	var names []string
	for name, register := range lx.registers {
		if _, good := register.(chan dslvm.Done); good {
			continue
		}
		logger.Warnf("register %s with type %T is not dropped: adding automatic drop", name, register)
		names = append(names, name)
	}
	return lx.addAutomaticDrop(logger, names...)
}

func (lx *loader) loadStages(logger model.Logger, stages ...StageNode) error {
	for _, entry := range stages {
		loader, good := lx.loaders[entry.Name]
		if !good {
			return fmt.Errorf("unknown instruction: %s", entry.Name)
		}
		if err := loader(entry.Value); err != nil {
			return err
		}
	}
	return nil
}

func (lx *loader) addAutomaticDrop(logger model.Logger, names ...string) error {
	for _, name := range names {
		err := lx.loadStages(logger, StageNode{
			Name: "drop",
			Value: must.MarshalJSON(dropValue{
				Input:  name,
				Output: name + "__autodrop",
			}),
		})
		if err != nil {
			return err
		}
	}
	return nil
}
