package webconnectivity_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"testing"

	"github.com/apex/log"
	"github.com/google/go-cmp/cmp"
	engine "github.com/ooni/probe-cli/v3/internal/engine"
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/webconnectivity"
	"github.com/ooni/probe-cli/v3/internal/engine/model"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/archival"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/errorx"
)

func TestNewExperimentMeasurer(t *testing.T) {
	measurer := webconnectivity.NewExperimentMeasurer(webconnectivity.Config{})
	if measurer.ExperimentName() != "web_connectivity" {
		t.Fatal("unexpected name")
	}
	if measurer.ExperimentVersion() != "0.4.0" {
		t.Fatal("unexpected version")
	}
}

func TestSuccess(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	measurer := webconnectivity.NewExperimentMeasurer(webconnectivity.Config{})
	ctx := context.Background()
	// we need a real session because we need the web-connectivity helper
	// as well as the ASN database
	sess := newsession(t, true)
	measurement := &model.Measurement{Input: "http://www.example.com"}
	callbacks := model.NewPrinterCallbacks(log.Log)
	err := measurer.Run(ctx, sess, measurement, callbacks)
	if err != nil {
		t.Fatal(err)
	}
	tk := measurement.TestKeys.(*webconnectivity.TestKeys)
	if tk.ControlFailure != nil {
		t.Fatal("unexpected control_failure")
	}
	if tk.DNSExperimentFailure != nil {
		t.Fatal("unexpected dns_experiment_failure")
	}
	if tk.HTTPExperimentFailure != nil {
		t.Fatal("unexpected http_experiment_failure")
	}
	// TODO(bassosimone): write further checks here?
}

func TestMeasureWithCancelledContext(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	measurer := webconnectivity.NewExperimentMeasurer(webconnectivity.Config{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // immediately fail
	// we need a real session because we need the web-connectivity helper
	sess := newsession(t, true)
	measurement := &model.Measurement{Input: "http://www.example.com"}
	callbacks := model.NewPrinterCallbacks(log.Log)
	if err := measurer.Run(ctx, sess, measurement, callbacks); err != nil {
		t.Fatal(err)
	}
	tk := measurement.TestKeys.(*webconnectivity.TestKeys)
	if *tk.ControlFailure != errorx.FailureInterrupted {
		t.Fatal("unexpected control_failure")
	}
	if *tk.DNSExperimentFailure != errorx.FailureInterrupted {
		t.Fatal("unexpected dns_experiment_failure")
	}
	if tk.HTTPExperimentFailure != nil {
		t.Fatal("unexpected http_experiment_failure")
	}
	// TODO(bassosimone): write further checks here?
	sk, err := measurer.GetSummaryKeys(measurement)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := sk.(webconnectivity.SummaryKeys); !ok {
		t.Fatal("invalid type for summary keys")
	}
}

func TestMeasureWithNoInput(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	measurer := webconnectivity.NewExperimentMeasurer(webconnectivity.Config{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// we need a real session because we need the web-connectivity helper
	sess := newsession(t, true)
	measurement := &model.Measurement{Input: ""}
	callbacks := model.NewPrinterCallbacks(log.Log)
	err := measurer.Run(ctx, sess, measurement, callbacks)
	if !errors.Is(err, webconnectivity.ErrNoInput) {
		t.Fatal(err)
	}
	tk := measurement.TestKeys.(*webconnectivity.TestKeys)
	if tk.ControlFailure != nil {
		t.Fatal("unexpected control_failure")
	}
	if tk.DNSExperimentFailure != nil {
		t.Fatal("unexpected dns_experiment_failure")
	}
	if tk.HTTPExperimentFailure != nil {
		t.Fatal("unexpected http_experiment_failure")
	}
	// TODO(bassosimone): write further checks here?
}

func TestMeasureWithInputNotBeingAnURL(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	measurer := webconnectivity.NewExperimentMeasurer(webconnectivity.Config{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// we need a real session because we need the web-connectivity helper
	sess := newsession(t, true)
	measurement := &model.Measurement{Input: "\t\t\t\t\t\t"}
	callbacks := model.NewPrinterCallbacks(log.Log)
	err := measurer.Run(ctx, sess, measurement, callbacks)
	if !errors.Is(err, webconnectivity.ErrInputIsNotAnURL) {
		t.Fatal(err)
	}
	tk := measurement.TestKeys.(*webconnectivity.TestKeys)
	if tk.ControlFailure != nil {
		t.Fatal("unexpected control_failure")
	}
	if tk.DNSExperimentFailure != nil {
		t.Fatal("unexpected dns_experiment_failure")
	}
	if tk.HTTPExperimentFailure != nil {
		t.Fatal("unexpected http_experiment_failure")
	}
	// TODO(bassosimone): write further checks here?
}

func TestMeasureWithUnsupportedInput(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	measurer := webconnectivity.NewExperimentMeasurer(webconnectivity.Config{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// we need a real session because we need the web-connectivity helper
	sess := newsession(t, true)
	measurement := &model.Measurement{Input: "dnslookup://example.com"}
	callbacks := model.NewPrinterCallbacks(log.Log)
	err := measurer.Run(ctx, sess, measurement, callbacks)
	if !errors.Is(err, webconnectivity.ErrUnsupportedInput) {
		t.Fatal(err)
	}
	tk := measurement.TestKeys.(*webconnectivity.TestKeys)
	if tk.ControlFailure != nil {
		t.Fatal("unexpected control_failure")
	}
	if tk.DNSExperimentFailure != nil {
		t.Fatal("unexpected dns_experiment_failure")
	}
	if tk.HTTPExperimentFailure != nil {
		t.Fatal("unexpected http_experiment_failure")
	}
	// TODO(bassosimone): write further checks here?
}

func TestMeasureWithNoAvailableTestHelpers(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	measurer := webconnectivity.NewExperimentMeasurer(webconnectivity.Config{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// we need a real session because we need the web-connectivity helper
	sess := newsession(t, false)
	measurement := &model.Measurement{Input: "https://www.example.com"}
	callbacks := model.NewPrinterCallbacks(log.Log)
	err := measurer.Run(ctx, sess, measurement, callbacks)
	if !errors.Is(err, webconnectivity.ErrNoAvailableTestHelpers) {
		t.Fatal(err)
	}
	tk := measurement.TestKeys.(*webconnectivity.TestKeys)
	if tk.ControlFailure != nil {
		t.Fatal("unexpected control_failure")
	}
	if tk.DNSExperimentFailure != nil {
		t.Fatal("unexpected dns_experiment_failure")
	}
	if tk.HTTPExperimentFailure != nil {
		t.Fatal("unexpected http_experiment_failure")
	}
	// TODO(bassosimone): write further checks here?
}

func newsession(t *testing.T, lookupBackends bool) model.ExperimentSession {
	sess, err := engine.NewSession(engine.SessionConfig{
		AvailableProbeServices: []model.Service{{
			Address: "https://ams-pg-test.ooni.org",
			Type:    "https",
		}},
		Logger:          log.Log,
		SoftwareName:    "ooniprobe-engine",
		SoftwareVersion: "0.0.1",
		TunnelDir:       filepath.Join(".", "testdata", "tunnel"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if lookupBackends {
		if err := sess.MaybeLookupBackends(); err != nil {
			t.Fatal(err)
		}
	}
	if err := sess.MaybeLookupLocation(); err != nil {
		t.Fatal(err)
	}
	return sess
}

func TestComputeTCPBlocking(t *testing.T) {
	var (
		falseValue = false
		trueValue  = true
	)
	failure := io.EOF.Error()
	anotherFailure := "unknown_error"
	type args struct {
		measurement []archival.TCPConnectEntry
		control     map[string]webconnectivity.ControlTCPConnectResult
	}
	tests := []struct {
		name string
		args args
		want []archival.TCPConnectEntry
	}{{
		name: "with all empty",
		args: args{},
		want: []archival.TCPConnectEntry{},
	}, {
		name: "with control failure",
		args: args{
			measurement: []archival.TCPConnectEntry{{
				IP:   "1.1.1.1",
				Port: 853,
				Status: archival.TCPConnectStatus{
					Failure: &failure,
					Success: false,
				},
			}},
		},
		want: []archival.TCPConnectEntry{{
			IP:   "1.1.1.1",
			Port: 853,
			Status: archival.TCPConnectStatus{
				Failure: &failure,
				Success: false,
			},
		}},
	}, {
		name: "with failures on both ends",
		args: args{
			measurement: []archival.TCPConnectEntry{{
				IP:   "1.1.1.1",
				Port: 853,
				Status: archival.TCPConnectStatus{
					Failure: &failure,
					Success: false,
				},
			}},
			control: map[string]webconnectivity.ControlTCPConnectResult{
				"1.1.1.1:853": {
					Failure: &anotherFailure,
					Status:  false,
				},
			},
		},
		want: []archival.TCPConnectEntry{{
			IP:   "1.1.1.1",
			Port: 853,
			Status: archival.TCPConnectStatus{
				Blocked: &falseValue,
				Failure: &failure,
				Success: false,
			},
		}},
	}, {
		name: "with failure on the probe side",
		args: args{
			measurement: []archival.TCPConnectEntry{{
				IP:   "1.1.1.1",
				Port: 853,
				Status: archival.TCPConnectStatus{
					Failure: &failure,
					Success: false,
				},
			}},
			control: map[string]webconnectivity.ControlTCPConnectResult{
				"1.1.1.1:853": {
					Failure: nil,
					Status:  true,
				},
			},
		},
		want: []archival.TCPConnectEntry{{
			IP:   "1.1.1.1",
			Port: 853,
			Status: archival.TCPConnectStatus{
				Blocked: &trueValue,
				Failure: &failure,
				Success: false,
			},
		}},
	}, {
		name: "with failure on the control side",
		args: args{
			measurement: []archival.TCPConnectEntry{{
				IP:   "1.1.1.1",
				Port: 853,
				Status: archival.TCPConnectStatus{
					Failure: nil,
					Success: true,
				},
			}},
			control: map[string]webconnectivity.ControlTCPConnectResult{
				"1.1.1.1:853": {
					Failure: &failure,
					Status:  false,
				},
			},
		},
		want: []archival.TCPConnectEntry{{
			IP:   "1.1.1.1",
			Port: 853,
			Status: archival.TCPConnectStatus{
				Blocked: &falseValue,
				Failure: nil,
				Success: true,
			},
		}},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := webconnectivity.ComputeTCPBlocking(tt.args.measurement, tt.args.control)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestSummaryKeysInvalidType(t *testing.T) {
	measurement := new(model.Measurement)
	m := &webconnectivity.Measurer{}
	_, err := m.GetSummaryKeys(measurement)
	if err.Error() != "invalid test keys type" {
		t.Fatal("not the error we expected")
	}
}

func TestSummaryKeysWorksAsIntended(t *testing.T) {
	failure := io.EOF.Error()
	truy := true
	tests := []struct {
		tk         webconnectivity.TestKeys
		Accessible bool
		Blocking   string
		isAnomaly  bool
	}{{
		tk:         webconnectivity.TestKeys{},
		Accessible: false,
		Blocking:   "",
		isAnomaly:  false,
	}, {
		tk: webconnectivity.TestKeys{Summary: webconnectivity.Summary{
			BlockingReason: &failure,
		}},
		Accessible: false,
		Blocking:   failure,
		isAnomaly:  true,
	}, {
		tk: webconnectivity.TestKeys{Summary: webconnectivity.Summary{
			Accessible: &truy,
		}},
		Accessible: true,
		Blocking:   "",
		isAnomaly:  false,
	}}
	for idx, tt := range tests {
		t.Run(fmt.Sprintf("%d", idx), func(t *testing.T) {
			m := &webconnectivity.Measurer{}
			measurement := &model.Measurement{TestKeys: &tt.tk}
			got, err := m.GetSummaryKeys(measurement)
			if err != nil {
				t.Fatal(err)
				return
			}
			sk := got.(webconnectivity.SummaryKeys)
			if sk.IsAnomaly != tt.isAnomaly {
				t.Fatal("unexpected isAnomaly value")
			}
			if sk.Accessible != tt.Accessible {
				t.Fatal("unexpected Accessible value")
			}
			if sk.Blocking != tt.Blocking {
				t.Fatal("unexpected Accessible value")
			}
		})
	}
}
