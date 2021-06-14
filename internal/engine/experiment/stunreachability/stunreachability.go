// Package stunreachability contains the STUN reachability experiment.
//
// See https://github.com/ooni/spec/blob/master/nettests/ts-025-stun-reachability.md.
package stunreachability

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine/model"
	"github.com/ooni/probe-cli/v3/internal/engine/netx"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/archival"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/dialer"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/trace"
	"github.com/pion/stun"
)

const (
	testName    = "stunreachability"
	testVersion = "0.2.0"
)

// Config contains the experiment config.
type Config struct {
	dialContext func(ctx context.Context, network, address string) (net.Conn, error)
	newClient   func(conn stun.Connection, options ...stun.ClientOption) (*stun.Client, error)
}

// TestKeys contains the experiment's result.
type TestKeys struct {
	Endpoint      string                   `json:"endpoint"`
	Failure       *string                  `json:"failure"`
	NetworkEvents []archival.NetworkEvent  `json:"network_events"`
	Queries       []archival.DNSQueryEntry `json:"queries"`
}

func registerExtensions(m *model.Measurement) {
	archival.ExtDNS.AddTo(m)
	archival.ExtNetevents.AddTo(m)
}

// Measurer performs the measurement.
type Measurer struct {
	config Config
}

// ExperimentName implements ExperimentMeasurer.ExperiExperimentName.
func (m *Measurer) ExperimentName() string {
	return testName
}

// ExperimentVersion implements ExperimentMeasurer.ExperimentVersion.
func (m *Measurer) ExperimentVersion() string {
	return testVersion
}

// Run implements ExperimentMeasurer.Run.
func (m *Measurer) Run(
	ctx context.Context, sess model.ExperimentSession,
	measurement *model.Measurement, callbacks model.ExperimentCallbacks,
) error {
	tk := new(TestKeys)
	measurement.TestKeys = tk
	registerExtensions(measurement)
	// TODO(kelmenhorst): wrap in Stun/TopLevel error wrapper
	if err := tk.run(ctx, m.config, sess, measurement, callbacks); err != nil {
		tk.Failure = archival.NewFailure(err)
		return err
	}
	return nil
}

func (tk *TestKeys) run(
	ctx context.Context, config Config, sess model.ExperimentSession,
	measurement *model.Measurement, callbacks model.ExperimentCallbacks,
) error {
	const defaultAddress = "stun.l.google.com:19302"
	endpoint := string(measurement.Input)
	if endpoint == "" {
		endpoint = defaultAddress
	}
	callbacks.OnProgress(0, fmt.Sprintf("stunreachability: measuring: %s...", endpoint))
	defer callbacks.OnProgress(
		1, fmt.Sprintf("stunreachability: measuring: %s... done", endpoint))
	tk.Endpoint = endpoint
	saver := new(trace.Saver)
	begin := time.Now()
	err := tk.do(ctx, config, netx.NewDialer(netx.Config{
		ContextByteCounting: true,
		DialSaver:           saver,
		Logger:              sess.Logger(),
		ReadWriteSaver:      saver,
		ResolveSaver:        saver,
	}), endpoint)
	events := saver.Read()
	tk.NetworkEvents = append(
		tk.NetworkEvents, archival.NewNetworkEventsList(begin, events)...,
	)
	tk.Queries = append(
		tk.Queries, archival.NewDNSQueriesList(begin, events)...,
	)
	return err
}

func (tk *TestKeys) do(
	ctx context.Context, config Config, dialer dialer.Dialer, endpoint string) error {
	dialContext := dialer.DialContext
	if config.dialContext != nil {
		dialContext = config.dialContext
	}
	conn, err := dialContext(ctx, "udp", endpoint)
	if err != nil {
		return err
	}
	newClient := stun.NewClient
	if config.newClient != nil {
		newClient = config.newClient
	}
	client, err := newClient(conn)
	if err != nil {
		return err
	}
	defer client.Close()
	message := stun.MustBuild(stun.TransactionID, stun.BindingRequest)
	ch := make(chan error)
	err = client.Start(message, func(ev stun.Event) {
		// As mentioned below this code will run after Start has returned.
		if ev.Error != nil {
			ch <- ev.Error
			return
		}
		var xorAddr stun.XORMappedAddress
		ch <- xorAddr.GetFrom(ev.Message)
	})
	// Implementation note: if we successfully started, then the callback
	// will be called when we receive a response or fail.
	if err != nil {
		return err
	}
	return <-ch
}

// NewExperimentMeasurer creates a new ExperimentMeasurer.
func NewExperimentMeasurer(config Config) model.ExperimentMeasurer {
	return &Measurer{config: config}
}

// SummaryKeys contains summary keys for this experiment.
//
// Note that this structure is part of the ABI contract with probe-cli
// therefore we should be careful when changing it.
type SummaryKeys struct {
	IsAnomaly bool `json:"-"`
}

// GetSummaryKeys implements model.ExperimentMeasurer.GetSummaryKeys.
func (m Measurer) GetSummaryKeys(measurement *model.Measurement) (interface{}, error) {
	return SummaryKeys{IsAnomaly: false}, nil
}
