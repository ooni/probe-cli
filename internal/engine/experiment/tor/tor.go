// Package tor contains the tor experiment.
//
// Spec: https://github.com/ooni/spec/blob/master/nettests/ts-023-tor.md
package tor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/ooni/probe-cli/v3/internal/atomicx"
	"github.com/ooni/probe-cli/v3/internal/engine/httpheader"
	"github.com/ooni/probe-cli/v3/internal/engine/legacy/netxlogger"
	"github.com/ooni/probe-cli/v3/internal/engine/legacy/oonidatamodel"
	"github.com/ooni/probe-cli/v3/internal/engine/legacy/oonitemplates"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/scrubber"
)

const (
	// parallelism is the number of parallel threads we use for this experiment
	parallelism = 2

	// testName is the name of this experiment
	testName = "tor"

	// testVersion is the version of this experiment
	testVersion = "0.3.0"
)

// Config contains the experiment config.
type Config struct{}

// Summary contains a summary of what happened.
type Summary struct {
	Failure *string `json:"failure"`
}

// TargetResults contains the results of measuring a target.
type TargetResults struct {
	Agent          string                          `json:"agent"`
	Failure        *string                         `json:"failure"`
	NetworkEvents  oonidatamodel.NetworkEventsList `json:"network_events"`
	Queries        oonidatamodel.DNSQueriesList    `json:"queries"`
	Requests       oonidatamodel.RequestList       `json:"requests"`
	Summary        map[string]Summary              `json:"summary"`
	TargetAddress  string                          `json:"target_address"`
	TargetName     string                          `json:"target_name,omitempty"`
	TargetProtocol string                          `json:"target_protocol"`
	TargetSource   string                          `json:"target_source,omitempty"`
	TCPConnect     oonidatamodel.TCPConnectList    `json:"tcp_connect"`
	TLSHandshakes  oonidatamodel.TLSHandshakesList `json:"tls_handshakes"`

	// Only for testing. We don't care about this field otherwise. We
	// cannot make this private because otherwise the IP address sanitizer
	// is going to panic over a private field.
	DirPortCount int `json:"-"`
}

func registerExtensions(m *model.Measurement) {
	oonidatamodel.ExtHTTP.AddTo(m)
	oonidatamodel.ExtNetevents.AddTo(m)
	oonidatamodel.ExtDNS.AddTo(m)
	oonidatamodel.ExtTCPConnect.AddTo(m)
	oonidatamodel.ExtTLSHandshake.AddTo(m)
}

// fillSummary fills the Summary field used by the UI.
func (tr *TargetResults) fillSummary() {
	tr.Summary = make(map[string]Summary)
	if len(tr.TCPConnect) < 1 {
		return
	}
	tr.Summary[netxlite.ConnectOperation] = Summary{
		Failure: tr.TCPConnect[0].Status.Failure,
	}
	switch tr.TargetProtocol {
	case "dir_port":
		// The UI currently doesn't care about this protocol
		// as long as drawing a table is concerned.
		tr.DirPortCount++
	case "obfs4":
		// We currently only perform an OBFS4 handshake, hence
		// the final Failure is the handshake result
		tr.Summary["handshake"] = Summary{
			Failure: tr.Failure,
		}
	case "or_port_dirauth", "or_port":
		if len(tr.TLSHandshakes) < 1 {
			return
		}
		tr.Summary["handshake"] = Summary{
			Failure: tr.TLSHandshakes[0].Failure,
		}
	}
}

// TestKeys contains tor test keys.
type TestKeys struct {
	DirPortTotal            int64                    `json:"dir_port_total"`
	DirPortAccessible       int64                    `json:"dir_port_accessible"`
	OBFS4Total              int64                    `json:"obfs4_total"`
	OBFS4Accessible         int64                    `json:"obfs4_accessible"`
	ORPortDirauthTotal      int64                    `json:"or_port_dirauth_total"`
	ORPortDirauthAccessible int64                    `json:"or_port_dirauth_accessible"`
	ORPortTotal             int64                    `json:"or_port_total"`
	ORPortAccessible        int64                    `json:"or_port_accessible"`
	Targets                 map[string]TargetResults `json:"targets"`
}

func (tk *TestKeys) fillToplevelKeys() {
	for _, value := range tk.Targets {
		switch value.TargetProtocol {
		case "dir_port":
			tk.DirPortTotal++
			if value.Failure == nil {
				tk.DirPortAccessible++
			}
		case "obfs4":
			tk.OBFS4Total++
			if value.Failure == nil {
				tk.OBFS4Accessible++
			}
		case "or_port_dirauth":
			tk.ORPortDirauthTotal++
			if value.Failure == nil {
				tk.ORPortDirauthAccessible++
			}
		case "or_port":
			tk.ORPortTotal++
			if value.Failure == nil {
				tk.ORPortAccessible++
			}
		}
	}
}

// Measurer performs the measurement.
type Measurer struct {
	config          Config
	fetchTorTargets func(ctx context.Context, sess model.ExperimentSession, cc string) (map[string]model.OOAPITorTarget, error)
}

// NewMeasurer creates a new Measurer
func NewMeasurer(config Config) *Measurer {
	return &Measurer{
		config: config,
		fetchTorTargets: func(ctx context.Context, sess model.ExperimentSession, cc string) (map[string]model.OOAPITorTarget, error) {
			return sess.FetchTorTargets(ctx, cc)
		},
	}
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
	ctx context.Context,
	sess model.ExperimentSession,
	measurement *model.Measurement,
	callbacks model.ExperimentCallbacks,
) error {
	targets, err := m.gimmeTargets(ctx, sess)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(
		ctx, 15*time.Second*time.Duration(len(targets)),
	)
	defer cancel()
	registerExtensions(measurement)
	m.measureTargets(ctx, sess, measurement, callbacks, targets)
	return nil
}

func (m *Measurer) gimmeTargets(
	ctx context.Context, sess model.ExperimentSession,
) (map[string]model.OOAPITorTarget, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	return m.fetchTorTargets(ctx, sess, sess.ProbeCC())
}

// keytarget contains a key and the related target
type keytarget struct {
	key    string
	target model.OOAPITorTarget
}

// private returns whether a target is private. We consider private
// every target coming from a non-empty data source.
func (kt keytarget) private() bool {
	return kt.target.Source != ""
}

// maybeTargetAddress returns the target address if the target is
// not private, otherwise it returns `"[scrubbed]""`.
func (kt keytarget) maybeTargetAddress() (address string) {
	address = "[scrubbed]"
	if !kt.private() {
		address = kt.target.Address
	}
	return
}

func (m *Measurer) measureTargets(
	ctx context.Context,
	sess model.ExperimentSession,
	measurement *model.Measurement,
	callbacks model.ExperimentCallbacks,
	targets map[string]model.OOAPITorTarget,
) {
	// run measurements in parallel
	var waitgroup sync.WaitGroup
	rc := newResultsCollector(sess, measurement, callbacks)
	waitgroup.Add(len(targets))
	workch := make(chan keytarget)
	for i := 0; i < parallelism; i++ {
		go func(ch <-chan keytarget, total int) {
			for kt := range ch {
				rc.measureSingleTarget(ctx, kt, total)
				waitgroup.Done()
			}
		}(workch, len(targets))
	}
	for key, target := range targets {
		workch <- keytarget{key: key, target: target}
	}
	close(workch)
	waitgroup.Wait()
	// fill the measurement entry
	testkeys := &TestKeys{Targets: rc.targetresults}
	testkeys.fillToplevelKeys()
	measurement.TestKeys = testkeys
}

type resultsCollector struct {
	callbacks       model.ExperimentCallbacks
	completed       *atomicx.Int64
	flexibleConnect func(context.Context, keytarget) (oonitemplates.Results, error)
	measurement     *model.Measurement
	mu              sync.Mutex
	sess            model.ExperimentSession
	targetresults   map[string]TargetResults
}

func newResultsCollector(
	sess model.ExperimentSession,
	measurement *model.Measurement,
	callbacks model.ExperimentCallbacks,
) *resultsCollector {
	rc := &resultsCollector{
		callbacks:     callbacks,
		completed:     &atomicx.Int64{},
		measurement:   measurement,
		sess:          sess,
		targetresults: make(map[string]TargetResults),
	}
	rc.flexibleConnect = rc.defaultFlexibleConnect
	return rc
}

func maybeSanitize(input TargetResults, kt keytarget) TargetResults {
	if !kt.private() {
		return input
	}
	data, err := json.Marshal(input)
	runtimex.PanicOnError(err, "json.Marshal should not fail here")
	// Implementation note: here we are using a strict scrubbing policy where
	// we remove all IP _endpoints_, mainly for convenience, because we already
	// have a well tested implementation that does that.
	data = []byte(scrubber.Scrub(string(data)))
	var out TargetResults
	err = json.Unmarshal(data, &out)
	runtimex.PanicOnError(err, "json.Unmarshal should not fail here")
	return out
}

func (rc *resultsCollector) measureSingleTarget(
	ctx context.Context, kt keytarget, total int,
) {
	tk, err := rc.flexibleConnect(ctx, kt)
	tr := TargetResults{
		Agent:         "redirect",
		Failure:       setFailure(err),
		NetworkEvents: oonidatamodel.NewNetworkEventsList(tk),
		Queries:       oonidatamodel.NewDNSQueriesList(tk),
		Requests:      oonidatamodel.NewRequestList(tk),
		TCPConnect:    oonidatamodel.NewTCPConnectList(tk),
		TLSHandshakes: oonidatamodel.NewTLSHandshakesList(tk),
	}
	tr.fillSummary()
	tr = maybeSanitize(tr, kt)
	rc.mu.Lock()
	tr.TargetAddress = kt.maybeTargetAddress()
	tr.TargetName = kt.target.Name
	tr.TargetProtocol = kt.target.Protocol
	tr.TargetSource = kt.target.Source
	rc.targetresults[kt.key] = tr
	rc.mu.Unlock()
	sofar := rc.completed.Add(1)
	percentage := 0.0
	if total > 0 {
		percentage = float64(sofar) / float64(total)
	}
	rc.callbacks.OnProgress(percentage, fmt.Sprintf(
		"tor: access %s/%s: %s", kt.maybeTargetAddress(), kt.target.Protocol,
		errString(err),
	))
}

func maybeScrubbingLogger(input model.Logger, kt keytarget) model.Logger {
	if !kt.private() {
		return input
	}
	return &scrubber.Logger{Logger: input}
}

func (rc *resultsCollector) defaultFlexibleConnect(
	ctx context.Context, kt keytarget,
) (tk oonitemplates.Results, err error) {
	logger := maybeScrubbingLogger(rc.sess.Logger(), kt)
	switch kt.target.Protocol {
	case "dir_port":
		url := url.URL{
			Host:   kt.target.Address,
			Path:   "/tor/status-vote/current/consensus.z",
			Scheme: "http",
		}
		const snapshotsize = 1 << 8 // no need to include all in report
		r := oonitemplates.HTTPDo(ctx, oonitemplates.HTTPDoConfig{
			Accept:                  httpheader.Accept(),
			AcceptLanguage:          httpheader.AcceptLanguage(),
			Beginning:               rc.measurement.MeasurementStartTimeSaved,
			MaxEventsBodySnapSize:   snapshotsize,
			MaxResponseBodySnapSize: snapshotsize,
			Handler:                 netxlogger.NewHandler(logger),
			Method:                  "GET",
			URL:                     url.String(),
			UserAgent:               httpheader.UserAgent(),
		})
		tk, err = r.TestKeys, r.Error
	case "or_port", "or_port_dirauth":
		r := oonitemplates.TLSConnect(ctx, oonitemplates.TLSConnectConfig{
			Address:            kt.target.Address,
			Beginning:          rc.measurement.MeasurementStartTimeSaved,
			InsecureSkipVerify: true,
			Handler:            netxlogger.NewHandler(logger),
		})
		tk, err = r.TestKeys, r.Error
	case "obfs4":
		r := oonitemplates.OBFS4Connect(ctx, oonitemplates.OBFS4ConnectConfig{
			Address:      kt.target.Address,
			Beginning:    rc.measurement.MeasurementStartTimeSaved,
			Handler:      netxlogger.NewHandler(logger),
			Params:       kt.target.Params,
			StateBaseDir: rc.sess.TempDir(),
		})
		tk, err = r.TestKeys, r.Error
	default:
		r := oonitemplates.TCPConnect(ctx, oonitemplates.TCPConnectConfig{
			Address:   kt.target.Address,
			Beginning: rc.measurement.MeasurementStartTimeSaved,
			Handler:   netxlogger.NewHandler(logger),
		})
		tk, err = r.TestKeys, r.Error
	}
	return
}

// NewExperimentMeasurer creates a new ExperimentMeasurer.
func NewExperimentMeasurer(config Config) model.ExperimentMeasurer {
	return NewMeasurer(config)
}

func errString(err error) (s string) {
	s = "success"
	if err != nil {
		s = err.Error()
	}
	return
}

func setFailure(err error) (s *string) {
	if err != nil {
		descr := err.Error()
		s = &descr
	}
	return
}

// SummaryKeys contains summary keys for this experiment.
//
// Note that this structure is part of the ABI contract with probe-cli
// therefore we should be careful when changing it.
type SummaryKeys struct {
	DirPortTotal            int64 `json:"dir_port_total"`
	DirPortAccessible       int64 `json:"dir_port_accessible"`
	OBFS4Total              int64 `json:"obfs4_total"`
	OBFS4Accessible         int64 `json:"obfs4_accessible"`
	ORPortDirauthTotal      int64 `json:"or_port_dirauth_total"`
	ORPortDirauthAccessible int64 `json:"or_port_dirauth_accessible"`
	ORPortTotal             int64 `json:"or_port_total"`
	ORPortAccessible        int64 `json:"or_port_accessible"`
	IsAnomaly               bool  `json:"-"`
}

// GetSummaryKeys implements model.ExperimentMeasurer.GetSummaryKeys.
func (m Measurer) GetSummaryKeys(measurement *model.Measurement) (interface{}, error) {
	sk := SummaryKeys{IsAnomaly: false}
	tk, ok := measurement.TestKeys.(*TestKeys)
	if !ok {
		return sk, errors.New("invalid test keys type")
	}
	sk.DirPortTotal = tk.DirPortTotal
	sk.DirPortAccessible = tk.DirPortAccessible
	sk.OBFS4Total = tk.OBFS4Total
	sk.OBFS4Accessible = tk.OBFS4Accessible
	sk.ORPortDirauthTotal = tk.ORPortDirauthTotal
	sk.ORPortDirauthAccessible = tk.ORPortDirauthAccessible
	sk.ORPortTotal = tk.ORPortTotal
	sk.ORPortAccessible = tk.ORPortAccessible
	sk.IsAnomaly = ((sk.DirPortAccessible <= 0 && sk.DirPortTotal > 0) ||
		(sk.OBFS4Accessible <= 0 && sk.OBFS4Total > 0) ||
		(sk.ORPortDirauthAccessible <= 0 && sk.ORPortDirauthTotal > 0) ||
		(sk.ORPortAccessible <= 0 && sk.ORPortTotal > 0))
	return sk, nil
}
