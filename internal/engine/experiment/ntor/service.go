package ntor

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/engine/model"
	"github.com/ooni/probe-cli/v3/internal/measuring/connector"
	"github.com/ooni/probe-cli/v3/internal/measuring/httptransport"
	"github.com/ooni/probe-cli/v3/internal/measuring/resolver"
	"github.com/ooni/probe-cli/v3/internal/measuring/tlshandshaker"
)

// serviceInput is the input for the measurement service.
type serviceInput struct {
	// name is the target name.
	name string

	// target contains the target info.
	target model.TorTarget
}

// serviceOutput is the output of the measurement service.
type serviceOutput struct {
	// results contains the target results
	results TargetResults
}

// service is the measurement service. The expected usage of
// this structure is the following:
//
// 1. call newService;
//
// 2. defer a call to svc.stop;
//
// 3. run svc.reader in a background goroutine;
//
// 4. read from svc.output.
type service struct {
	// connector is the connector service.
	connector *connector.Service

	// httpTransport is the HTTP transport service.
	httpTransport *httptransport.Service

	// input is the input channel.
	input chan *serviceInput

	// output is the output channel.
	output chan *serviceOutput

	// resolver is the resolver service.
	resolver *resolver.Service

	// tlsHandshaker is the TLS handshaker service.
	tlsHandshaker *tlshandshaker.Service
}

// newService creates a new measurement service. This method will:
//
// 1. initialize the new service;
//
// 2. start a bunch of goroutines for performing measurements;
//
// 3. start all the required child services.
func newService(ctx context.Context) *service {
	svc := &service{
		connector:     connector.New(),
		httpTransport: httptransport.New(),
		input:         make(chan *serviceInput),
		output:        make(chan *serviceOutput),
		resolver:      resolver.New(),
		tlsHandshaker: tlshandshaker.New(),
	}
	const parallelism = 10
	for i := 0; i < parallelism; i++ {
		go svc.workerloop(ctx)
	}
	// note: we use less parallelism for heavier operations
	svc.connector.StartN(10)
	svc.httpTransport.StartN(2)
	svc.resolver.StartN(10)
	svc.tlsHandshaker.StartN(4)
	return svc
}

// stop stops all the child services managed by measurementCtx.
func (svc *service) stop() {
	svc.connector.Stop()
	svc.httpTransport.Stop()
	svc.resolver.Stop()
	svc.tlsHandshaker.Stop()
}

// reader reads and dispatches inputs asynchronously to the
// background goroutines. When done, the reader will close
// the svc.input channel and terminate.
func (svc *service) reader(targets map[string]model.TorTarget) {
	for name, info := range targets {
		svc.input <- &serviceInput{
			name:   name,
			target: info,
		}
	}
	close(svc.input) // tell EOF to the workers
}

// workerloop runs the service-worker's loop.
func (svc *service) workerloop(ctx context.Context) {
	for input := range svc.input {
		// TODO(bassosimone): replace this stub w/ real code
		svc.output <- &serviceOutput{
			results: TargetResults{
				TargetAddress:  input.target.Address,
				TargetName:     input.name,
				TargetProtocol: input.target.Protocol,
				TargetSource:   input.target.Source,
			},
		}
	}
}
