package tor

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/apex/log"
	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/engine/mockable"
	"github.com/ooni/probe-cli/v3/internal/measurex"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/scrubber"
)

func TestNewExperimentMeasurer(t *testing.T) {
	measurer := NewExperimentMeasurer(Config{})
	if measurer.ExperimentName() != "tor" {
		t.Fatal("unexpected name")
	}
	if measurer.ExperimentVersion() != "0.4.0" {
		t.Fatal("unexpected version")
	}
}

func TestMeasurerMeasureFetchTorTargetsError(t *testing.T) {
	measurer := NewMeasurer(Config{})
	expected := errors.New("mocked error")
	measurer.fetchTorTargets = func(ctx context.Context, sess model.ExperimentSession, cc string) (map[string]model.OOAPITorTarget, error) {
		return nil, expected
	}
	args := &model.ExperimentArgs{
		Callbacks:   model.NewPrinterCallbacks(log.Log),
		Measurement: &model.Measurement{},
		Session: &mockable.Session{
			MockableLogger: log.Log,
		},
	}
	err := measurer.Run(context.Background(), args)
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected")
	}
}

func TestMeasurerMeasureFetchTorTargetsEmptyList(t *testing.T) {
	measurer := NewMeasurer(Config{})
	measurer.fetchTorTargets = func(ctx context.Context, sess model.ExperimentSession, cc string) (map[string]model.OOAPITorTarget, error) {
		return nil, nil
	}
	measurement := new(model.Measurement)
	args := &model.ExperimentArgs{
		Callbacks:   model.NewPrinterCallbacks(log.Log),
		Measurement: measurement,
		Session: &mockable.Session{
			MockableLogger: log.Log,
		},
	}
	err := measurer.Run(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	tk := measurement.TestKeys.(*TestKeys)
	if len(tk.Targets) != 0 {
		t.Fatal("expected no targets here")
	}
}

func TestMeasurerMeasureGoodWithMockedOrchestra(t *testing.T) {
	// This test mocks orchestra to return a nil list of targets, so the code runs
	// but we don't perform any actual network actions.
	measurer := NewMeasurer(Config{})
	measurer.fetchTorTargets = func(ctx context.Context, sess model.ExperimentSession, cc string) (map[string]model.OOAPITorTarget, error) {
		return nil, nil
	}
	args := &model.ExperimentArgs{
		Callbacks:   model.NewPrinterCallbacks(log.Log),
		Measurement: &model.Measurement{},
		Session: &mockable.Session{
			MockableLogger: log.Log,
		},
	}
	err := measurer.Run(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
}

func TestMeasurerMeasureGood(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	measurer := NewMeasurer(Config{})
	sess := newsession()
	measurement := new(model.Measurement)
	args := &model.ExperimentArgs{
		Callbacks:   model.NewPrinterCallbacks(log.Log),
		Measurement: measurement,
		Session:     sess,
	}
	err := measurer.Run(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	sk, err := measurer.GetSummaryKeys(measurement)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := sk.(SummaryKeys); !ok {
		t.Fatal("invalid type for summary keys")
	}
}

var staticPrivateTestingTargetEndpoint = "209.148.46.65:443"

var staticPrivateTestingTarget = model.OOAPITorTarget{
	Address: staticPrivateTestingTargetEndpoint,
	Params: map[string][]string{
		"cert": {
			"ssH+9rP8dG2NLDN2XuFw63hIO/9MNNinLmxQDpVa+7kTOa9/m+tGWT1SmSYpQ9uTBGa6Hw",
		},
		"iat-mode": {"0"},
	},
	Protocol: "obfs4",
	Source:   "bridgedb",
}

func TestMeasurerMeasureSanitiseOutput(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	measurer := NewMeasurer(Config{})
	sess := newsession()
	key := "xyz-xyz-xyz-theCh2ju-ahG4chei-Ai2eka0a"
	sess.MockableFetchTorTargetsResult = map[string]model.OOAPITorTarget{
		key: staticPrivateTestingTarget,
	}
	measurement := new(model.Measurement)
	args := &model.ExperimentArgs{
		Callbacks:   model.NewPrinterCallbacks(log.Log),
		Measurement: measurement,
		Session:     sess,
	}
	err := measurer.Run(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(measurement)
	if err != nil {
		t.Fatal(err)
	}
	tk := measurement.TestKeys.(*TestKeys)
	entry := tk.Targets[key]
	if entry.Failure != nil {
		t.Fatal("measurement failed unexpectedly", *entry.Failure)
	}
	if !bytes.Contains(data, []byte(key)) {
		t.Fatal("cannot find expected key")
	}
	if bytes.Contains(data, []byte(staticPrivateTestingTargetEndpoint)) {
		t.Fatal("endpoint found in serialized measurement")
	}
	if !bytes.Contains(data, []byte("[scrubbed]")) {
		t.Fatal("[scrubbed] not found in serialized measurement")
	}
}

var staticTestingTargets = []model.OOAPITorTarget{
	{
		Address: "192.95.36.142:443",
		Params: map[string][]string{
			"cert": {
				"qUVQ0srL1JI/vO6V6m/24anYXiJD3QP2HgzUKQtQ7GRqqUvs7P+tG43RtAqdhLOALP7DJQ",
			},
			"iat-mode": {"1"},
		},
		Protocol: "obfs4",
	},
	{
		Address:  "66.111.2.131:9030",
		Protocol: "dir_port",
	},
	{
		Address:  "66.111.2.131:9001",
		Protocol: "or_port",
	},
	{
		Address:  "1.1.1.1:80",
		Protocol: "tcp",
	},
}

func TestMeasurerMeasureTargetsNoInput(t *testing.T) {
	var measurement model.Measurement
	measurer := new(Measurer)
	measurer.measureTargets(
		context.Background(),
		&mockable.Session{
			MockableLogger: log.Log,
		},
		&measurement,
		model.NewPrinterCallbacks(log.Log),
		nil,
	)
	if len(measurement.TestKeys.(*TestKeys).Targets) != 0 {
		t.Fatal("expected no measurements here")
	}
}

func TestMeasurerMeasureTargetsCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // so we don't actually do anything
	var measurement model.Measurement
	measurer := new(Measurer)
	measurer.measureTargets(
		ctx,
		&mockable.Session{
			MockableLogger: log.Log,
		},
		&measurement,
		model.NewPrinterCallbacks(log.Log),
		map[string]model.OOAPITorTarget{
			"xx": staticTestingTargets[0],
		},
	)
	targets := measurement.TestKeys.(*TestKeys).Targets
	if len(targets) != 1 {
		t.Fatal("expected single measurements here")
	}
	if _, found := targets["xx"]; !found {
		t.Fatal("the target we expected is missing")
	}
	tgt := targets["xx"]
	if *tgt.Failure != "interrupted" {
		t.Fatal("not the error we expected")
	}
}

func wrapTestingTarget(tt model.OOAPITorTarget) keytarget {
	return keytarget{
		key:    "xx", // using an super simple key; should work anyway
		target: tt,
	}
}

func TestResultsCollectorMeasureSingleTargetGood(t *testing.T) {
	rc := newResultsCollector(
		&mockable.Session{
			MockableLogger: log.Log,
		},
		new(model.Measurement),
		model.NewPrinterCallbacks(log.Log),
	)
	rc.flexibleConnect = func(context.Context, keytarget) (*measurex.ArchivalMeasurement, *string) {
		return &measurex.ArchivalMeasurement{}, nil
	}
	rc.measureSingleTarget(
		context.Background(), wrapTestingTarget(staticTestingTargets[0]),
		len(staticTestingTargets),
	)
	if len(rc.targetresults) != 1 {
		t.Fatal("wrong number of entries")
	}
	// Implementation note: here we won't bother with checking that
	// oonidatamodel works correctly because we already test that.
	if rc.targetresults["xx"].Agent != "redirect" {
		t.Fatal("agent is invalid")
	}
	if rc.targetresults["xx"].Failure != nil {
		t.Fatal("failure is invalid")
	}
	if rc.targetresults["xx"].TargetAddress != staticTestingTargets[0].Address {
		t.Fatal("target address is invalid")
	}
	if rc.targetresults["xx"].TargetProtocol != staticTestingTargets[0].Protocol {
		t.Fatal("target protocol is invalid")
	}
}

func TestResultsCollectorMeasureSingleTargetWithFailure(t *testing.T) {
	rc := newResultsCollector(
		&mockable.Session{
			MockableLogger: log.Log,
		},
		new(model.Measurement),
		model.NewPrinterCallbacks(log.Log),
	)
	rc.flexibleConnect = func(context.Context, keytarget) (*measurex.ArchivalMeasurement, *string) {
		failure := "mocked error"
		return &measurex.ArchivalMeasurement{}, &failure
	}
	rc.measureSingleTarget(
		context.Background(), keytarget{
			key:    "xx", // using an super simple key; should work anyway
			target: staticTestingTargets[0],
		},
		len(staticTestingTargets),
	)
	if len(rc.targetresults) != 1 {
		t.Fatal("wrong number of entries")
	}
	// Implementation note: here we won't bother with checking that
	// oonidatamodel works correctly because we already test that.
	if rc.targetresults["xx"].Agent != "redirect" {
		t.Fatal("agent is invalid")
	}
	if *rc.targetresults["xx"].Failure != "mocked error" {
		t.Fatal("failure is invalid")
	}
	if rc.targetresults["xx"].TargetAddress != staticTestingTargets[0].Address {
		t.Fatal("target address is invalid")
	}
	if rc.targetresults["xx"].TargetProtocol != staticTestingTargets[0].Protocol {
		t.Fatal("target protocol is invalid")
	}
}

func TestDefautFlexibleConnectDirPort(t *testing.T) {
	rc := newResultsCollector(
		&mockable.Session{
			MockableLogger: log.Log,
		},
		new(model.Measurement),
		model.NewPrinterCallbacks(log.Log),
	)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	tk, failure := rc.defaultFlexibleConnect(ctx, wrapTestingTarget(staticTestingTargets[1]))
	if failure == nil {
		t.Fatal("expected a failure here")
	}
	if !strings.HasSuffix(*failure, "interrupted") {
		t.Fatal("not the error we expected")
	}
	if tk.Requests == nil {
		t.Fatal("expected HTTP data here")
	}
}

func TestDefautFlexibleConnectOrPort(t *testing.T) {
	rc := newResultsCollector(
		&mockable.Session{
			MockableLogger: log.Log,
		},
		new(model.Measurement),
		model.NewPrinterCallbacks(log.Log),
	)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	tk, failure := rc.defaultFlexibleConnect(ctx, wrapTestingTarget(staticTestingTargets[2]))
	if failure == nil {
		t.Fatal("expected a failure here")
	}
	if *failure != "interrupted" {
		t.Fatal("not the error we expected")
	}
	if tk.TCPConnect == nil {
		t.Fatal("expected connects data here")
	}
	if tk.NetworkEvents != nil {
		t.Fatal("expected no network events data here")
	}
}

func TestDefautFlexibleConnectOBFS4(t *testing.T) {
	rc := newResultsCollector(
		&mockable.Session{
			MockableLogger: log.Log,
		},
		new(model.Measurement),
		model.NewPrinterCallbacks(log.Log),
	)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	tk, failure := rc.defaultFlexibleConnect(ctx, wrapTestingTarget(staticTestingTargets[0]))
	if failure == nil {
		t.Fatal("expected a failure here")
	}
	if *failure != "interrupted" {
		t.Fatal("not the error we expected")
	}
	if tk.TCPConnect == nil {
		t.Fatal("expected connects data here")
	}
	if tk.NetworkEvents != nil {
		t.Fatal("expected no network events data here")
	}
}

func TestDefautFlexibleConnectDefault(t *testing.T) {
	rc := newResultsCollector(
		&mockable.Session{
			MockableLogger: log.Log,
		},
		new(model.Measurement),
		model.NewPrinterCallbacks(log.Log),
	)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	tk, failure := rc.defaultFlexibleConnect(ctx, wrapTestingTarget(staticTestingTargets[3]))
	if failure == nil {
		t.Fatal("expected a failure here")
	}
	if *failure != "interrupted" {
		t.Fatalf("not the error we expected: %+v", *failure)
	}
	if tk.TCPConnect == nil {
		t.Fatalf("expected connects data here, found: %+v", tk.TCPConnect)
	}
}

func TestFailureString(t *testing.T) {
	if failureString(nil) != "success" {
		t.Fatal("not working with nil")
	}
	s := "antani"
	if failureString(&s) != "antani" {
		t.Fatal("not working with non-nil string")
	}
}

func TestSummary(t *testing.T) {
	t.Run("without any piece of data", func(t *testing.T) {
		tr := new(TargetResults)
		tr.fillSummary()
		if len(tr.Summary) != 0 {
			t.Fatal("summary must be empty")
		}
	})

	t.Run("with a TCP connect and nothing else", func(t *testing.T) {
		tr := new(TargetResults)
		failure := "mocked_error"
		tr.TCPConnect = append(tr.TCPConnect, &measurex.ArchivalTCPConnect{
			Status: &measurex.ArchivalTCPConnectStatus{
				Success: true,
				Failure: &failure,
			},
		})
		tr.fillSummary()
		if len(tr.Summary) != 1 {
			t.Fatal("cannot find expected entry")
		}
		if *tr.Summary[netxlite.ConnectOperation].Failure != failure {
			t.Fatal("invalid failure")
		}
	})

	t.Run("for OBFS4", func(t *testing.T) {
		tr := new(TargetResults)
		tr.TCPConnect = append(tr.TCPConnect, &measurex.ArchivalTCPConnect{
			Status: &measurex.ArchivalTCPConnectStatus{
				Success: true,
			},
		})
		failure := "mocked_error"
		tr.TargetProtocol = "obfs4"
		tr.Failure = &failure
		tr.fillSummary()
		if len(tr.Summary) != 2 {
			t.Fatal("cannot find expected entry")
		}
		if tr.Summary[netxlite.ConnectOperation].Failure != nil {
			t.Fatal("invalid failure")
		}
		if *tr.Summary["handshake"].Failure != failure {
			t.Fatal("invalid failure")
		}
	})

	t.Run("for or_port/or_port_dirauth", func(t *testing.T) {
		doit := func(targetProtocol string, handshake *measurex.ArchivalQUICTLSHandshakeEvent) {
			tr := new(TargetResults)
			tr.TCPConnect = append(tr.TCPConnect, &measurex.ArchivalTCPConnect{
				Status: &measurex.ArchivalTCPConnectStatus{
					Success: true,
				},
			})
			tr.TargetProtocol = targetProtocol
			if handshake != nil {
				tr.TLSHandshakes = append(tr.TLSHandshakes, handshake)
			}
			tr.fillSummary()
			if len(tr.Summary) < 1 {
				t.Fatal("cannot find expected entry")
			}
			if tr.Summary[netxlite.ConnectOperation].Failure != nil {
				t.Fatal("invalid failure")
			}
			if handshake == nil {
				if len(tr.Summary) != 1 {
					t.Fatal("unexpected summary length")
				}
				return
			}
			if len(tr.Summary) != 2 {
				t.Fatal("unexpected summary length")
			}
			if tr.Summary["handshake"].Failure != handshake.Failure {
				t.Fatal("the failure value is unexpected")
			}
		}
		doit("or_port_dirauth", nil)
		doit("or_port", nil)
		doit("or_port", &measurex.ArchivalQUICTLSHandshakeEvent{
			Failure: (func() *string {
				s := io.EOF.Error()
				return &s
			})(),
		})
	})
}

func TestFillToplevelKeys(t *testing.T) {
	var tr TargetResults
	tr.TargetProtocol = "or_port"
	tk := new(TestKeys)
	tk.Targets = make(map[string]TargetResults)
	tk.Targets["xxx"] = tr
	tk.fillToplevelKeys()
	if tk.ORPortTotal != 1 {
		t.Fatal("unexpected ORPortTotal value")
	}
}

func newsession() *mockable.Session {
	return &mockable.Session{
		MockableLogger:     log.Log,
		MockableHTTPClient: http.DefaultClient,
	}
}

var referenceTargetResult = []byte(`{
	"agent": "redirect",
	"failure": null,
	"network_events": [
	  {
		"address": "85.31.186.98:443",
		"conn_id": 19,
		"dial_id": 21,
		"failure": null,
		"operation": "connect",
		"proto": "tcp",
		"t": 8.639313
	  },
	  {
		"conn_id": 19,
		"failure": null,
		"num_bytes": 1915,
		"operation": "write",
		"proto": "tcp",
		"t": 8.639686
	  },
	  {
		"conn_id": 19,
		"failure": null,
		"num_bytes": 1440,
		"operation": "read",
		"proto": "tcp",
		"t": 8.691708
	  },
	  {
		"conn_id": 19,
		"failure": null,
		"num_bytes": 1440,
		"operation": "read",
		"proto": "tcp",
		"t": 8.691912
	  },
	  {
		"conn_id": 19,
		"failure": null,
		"num_bytes": 1383,
		"operation": "read",
		"proto": "tcp",
		"t": 8.69234
	  }
	],
	"queries": null,
	"requests": null,
	"summary": {
	  "connect": {
		"failure": null
	  }
	},
	"target_address": "85.31.186.98:443",
	"target_protocol": "obfs4",
	"tcp_connect": [
	  {
		"conn_id": 19,
		"dial_id": 21,
		"ip": "85.31.186.98",
		"port": 443,
		"status": {
		  "failure": null,
		  "success": true
		},
		"t": 8.639313
	  }
	],
	"tls_handshakes": null
  }`)

var scrubbedTargetResult = []byte(`{
	"agent": "redirect",
	"failure": null,
	"network_events": [
	  {
		"address": "[scrubbed]",
		"conn_id": 19,
		"dial_id": 21,
		"failure": null,
		"operation": "connect",
		"proto": "tcp",
		"t": 8.639313
	  },
	  {
		"conn_id": 19,
		"failure": null,
		"num_bytes": 1915,
		"operation": "write",
		"proto": "tcp",
		"t": 8.639686
	  },
	  {
		"conn_id": 19,
		"failure": null,
		"num_bytes": 1440,
		"operation": "read",
		"proto": "tcp",
		"t": 8.691708
	  },
	  {
		"conn_id": 19,
		"failure": null,
		"num_bytes": 1440,
		"operation": "read",
		"proto": "tcp",
		"t": 8.691912
	  },
	  {
		"conn_id": 19,
		"failure": null,
		"num_bytes": 1383,
		"operation": "read",
		"proto": "tcp",
		"t": 8.69234
	  }
	],
	"queries": null,
	"requests": null,
	"summary": {
	  "connect": {
		"failure": null
	  }
	},
	"target_address": "[scrubbed]",
	"target_protocol": "obfs4",
	"tcp_connect": [
	  {
		"conn_id": 19,
		"dial_id": 21,
		"ip": "[scrubbed]",
		"port": 443,
		"status": {
		  "failure": null,
		  "success": true
		},
		"t": 8.639313
	  }
	],
	"tls_handshakes": null
  }`)

func TestMaybeSanitize(t *testing.T) {
	var input TargetResults
	if err := json.Unmarshal(referenceTargetResult, &input); err != nil {
		t.Fatal(err)
	}
	t.Run("nothing to do", func(t *testing.T) {
		out := maybeSanitize(input, keytarget{target: model.OOAPITorTarget{Source: ""}})
		diff := cmp.Diff(input, out)
		if diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("scrubbing to do", func(t *testing.T) {
		var expected TargetResults
		if err := json.Unmarshal(scrubbedTargetResult, &expected); err != nil {
			t.Fatal(err)
		}
		out := maybeSanitize(input, keytarget{target: model.OOAPITorTarget{
			Address: "85.31.186.98:443",
			Source:  "bridgedb",
		}})
		diff := cmp.Diff(expected, out)
		if diff != "" {
			t.Fatal(diff)
		}
	})
}

func TestMaybeScrubbingLogger(t *testing.T) {
	var input model.Logger = log.Log

	t.Run("for when we don't need to save", func(t *testing.T) {
		kt := keytarget{target: model.OOAPITorTarget{
			Source: "",
		}}
		out := maybeScrubbingLogger(input, kt)
		if out != input {
			t.Fatal("not the output we expected")
		}
		if _, ok := out.(*scrubber.Logger); ok {
			t.Fatal("not the output type we expected")
		}
	})

	t.Run("for when we need to save", func(t *testing.T) {
		kt := keytarget{target: model.OOAPITorTarget{
			Source: "bridgedb",
		}}
		out := maybeScrubbingLogger(input, kt)
		if out == input {
			t.Fatal("not the output value we expected")
		}
		if _, ok := out.(*scrubber.Logger); !ok {
			t.Fatal("not the output type we expected")
		}
	})
}

func TestSummaryKeysInvalidType(t *testing.T) {
	measurement := new(model.Measurement)
	m := &Measurer{}
	_, err := m.GetSummaryKeys(measurement)
	if err.Error() != "invalid test keys type" {
		t.Fatal("not the error we expected")
	}
}

func TestSummaryKeysWorksAsIntended(t *testing.T) {
	tests := []struct {
		tk        TestKeys
		isAnomaly bool
	}{{
		tk:        TestKeys{},
		isAnomaly: false,
	}, {
		tk:        TestKeys{DirPortAccessible: 1, DirPortTotal: 3},
		isAnomaly: false,
	}, {
		tk:        TestKeys{DirPortAccessible: 0, DirPortTotal: 3},
		isAnomaly: true,
	}, {
		tk:        TestKeys{OBFS4Accessible: 1, OBFS4Total: 3},
		isAnomaly: false,
	}, {
		tk:        TestKeys{OBFS4Accessible: 0, OBFS4Total: 3},
		isAnomaly: true,
	}, {
		tk:        TestKeys{ORPortDirauthAccessible: 1, ORPortDirauthTotal: 3},
		isAnomaly: false,
	}, {
		tk:        TestKeys{ORPortDirauthAccessible: 0, ORPortDirauthTotal: 3},
		isAnomaly: true,
	}, {
		tk:        TestKeys{ORPortAccessible: 1, ORPortTotal: 3},
		isAnomaly: false,
	}, {
		tk:        TestKeys{ORPortAccessible: 0, ORPortTotal: 3},
		isAnomaly: true,
	}}
	for idx, tt := range tests {
		t.Run(fmt.Sprintf("%d", idx), func(t *testing.T) {
			m := &Measurer{}
			measurement := &model.Measurement{TestKeys: &tt.tk}
			got, err := m.GetSummaryKeys(measurement)
			if err != nil {
				t.Fatal(err)
				return
			}
			sk := got.(SummaryKeys)
			if sk.IsAnomaly != tt.isAnomaly {
				t.Fatal("unexpected isAnomaly value")
			}
		})
	}
}

func TestTargetResultsFillSummaryDirPort(t *testing.T) {
	tr := &TargetResults{
		TargetProtocol: "dir_port",
		TCPConnect: []*measurex.ArchivalTCPConnect{{
			IP:   "1.2.3.4",
			Port: 443,
			Status: &measurex.ArchivalTCPConnectStatus{
				Failure: nil,
			},
		}},
	}
	tr.fillSummary()
	if tr.DirPortCount != 1 {
		t.Fatal("unexpected dirPortCount")
	}
}

func TestTestKeysFillToplevelKeysCoverMissingFields(t *testing.T) {
	failureString := "eof_error"
	tk := &TestKeys{
		Targets: map[string]TargetResults{
			"foobar":  {Failure: &failureString, TargetProtocol: "dir_port"},
			"baz":     {TargetProtocol: "dir_port"},
			"jafar":   {Failure: &failureString, TargetProtocol: "or_port_dirauth"},
			"jasmine": {TargetProtocol: "or_port_dirauth"},
		},
	}
	tk.fillToplevelKeys()
	if tk.DirPortTotal != 2 {
		t.Fatal("unexpected DirPortTotal")
	}
	if tk.DirPortAccessible != 1 {
		t.Fatal("unexpected DirPortAccessible")
	}
	if tk.ORPortDirauthTotal != 2 {
		t.Fatal("unexpected ORPortDirauthTotal")
	}
	if tk.ORPortDirauthAccessible != 1 {
		t.Fatal("unexpected ORPortDirauthAccessible")
	}
}
