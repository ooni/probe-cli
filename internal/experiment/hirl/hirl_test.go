package hirl_test

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/experiment/hirl"
	"github.com/ooni/probe-cli/v3/internal/legacy/mockable"
	"github.com/ooni/probe-cli/v3/internal/legacy/netx"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/tracex"
)

func TestNewExperimentMeasurer(t *testing.T) {
	measurer := hirl.NewExperimentMeasurer(hirl.Config{})
	if measurer.ExperimentName() != "http_invalid_request_line" {
		t.Fatal("unexpected name")
	}
	if measurer.ExperimentVersion() != "0.2.1" {
		t.Fatal("unexpected version")
	}
}

func TestSuccess(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	measurer := hirl.NewExperimentMeasurer(hirl.Config{})
	ctx := context.Background()
	sess := &mockable.Session{
		MockableLogger: log.Log,
		MockableTestHelpers: map[string][]model.OOAPIService{
			"tcp-echo": {{
				Address: "37.218.241.93",
				Type:    "legacy",
			}},
		},
	}
	measurement := new(model.Measurement)
	callbacks := model.NewPrinterCallbacks(log.Log)
	args := &model.ExperimentArgs{
		Callbacks:   callbacks,
		Measurement: measurement,
		Session:     sess,
	}
	err := measurer.Run(ctx, args)
	if err != nil {
		t.Fatal(err)
	}
	tk := measurement.TestKeys.(*hirl.TestKeys)
	if len(tk.FailureList) != len(tk.Received) {
		t.Fatal("FailureList and Received have different lengths")
	}
	if len(tk.Received) != len(tk.Sent) {
		t.Fatal("Received and Sent have different lengths")
	}
	if len(tk.Sent) != len(tk.TamperingList) {
		t.Fatal("Sent and TamperingList have different lengths")
	}
	for _, failure := range tk.FailureList {
		if failure != nil {
			t.Fatal(*failure)
		}
	}
	for idx, received := range tk.Received {
		if received.Value != tk.Sent[idx] {
			t.Fatal("mismatch between received and sent")
		}
	}
	for _, entry := range tk.TamperingList {
		if entry != false {
			t.Fatal("found entry with tampering")
		}
	}
	if tk.Tampering != false {
		t.Fatal("overall there is tampering?!")
	}
}

func TestCancelledContext(t *testing.T) {
	measurer := hirl.NewExperimentMeasurer(hirl.Config{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	sess := &mockable.Session{
		MockableLogger: log.Log,
		MockableTestHelpers: map[string][]model.OOAPIService{
			"tcp-echo": {{
				Address: "37.218.241.93",
				Type:    "legacy",
			}},
		},
	}
	measurement := new(model.Measurement)
	callbacks := model.NewPrinterCallbacks(log.Log)
	args := &model.ExperimentArgs{
		Callbacks:   callbacks,
		Measurement: measurement,
		Session:     sess,
	}
	err := measurer.Run(ctx, args)
	if err != nil {
		t.Fatal(err)
	}
	tk := measurement.TestKeys.(*hirl.TestKeys)
	if len(tk.FailureList) != 5 {
		t.Fatal("unexpected FailureList length")
	}
	for _, failure := range tk.FailureList {
		if *failure != netxlite.FailureInterrupted {
			t.Fatal("unexpected failure")
		}
	}
	if len(tk.Received) != 5 {
		t.Fatal("unexpected Received length")
	}
	for _, entry := range tk.Received {
		if entry.Value != "" {
			t.Fatal("unexpected received entry")
		}
	}
	if len(tk.Sent) != 5 {
		t.Fatal("unexpected Sent length")
	}
	for _, entry := range tk.Sent {
		if entry != "" {
			t.Fatal("unexpected sent entry")
		}
	}
	if len(tk.TamperingList) != 5 {
		t.Fatal("unexpected TamperingList length")
	}
	for _, entry := range tk.TamperingList {
		if entry != false {
			t.Fatal("unexpected tampering entry")
		}
	}
	if tk.Tampering != false {
		t.Fatal("overall there is tampering?!")
	}
	sk, err := measurer.GetSummaryKeys(measurement)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := sk.(hirl.SummaryKeys); !ok {
		t.Fatal("invalid type for summary keys")
	}
}

type FakeMethodSuccessful struct{}

func (FakeMethodSuccessful) Name() string {
	return "success"
}

func (meth FakeMethodSuccessful) Run(ctx context.Context, config hirl.MethodConfig) {
	config.Out <- hirl.MethodResult{
		Name:      meth.Name(),
		Received:  tracex.MaybeBinaryValue{Value: "antani"},
		Sent:      "antani",
		Tampering: false,
	}
}

type FakeMethodFailure struct{}

func (FakeMethodFailure) Name() string {
	return "failure"
}

func (meth FakeMethodFailure) Run(ctx context.Context, config hirl.MethodConfig) {
	config.Out <- hirl.MethodResult{
		Name:      meth.Name(),
		Received:  tracex.MaybeBinaryValue{Value: "antani"},
		Sent:      "melandri",
		Tampering: true,
	}
}

func TestWithFakeMethods(t *testing.T) {
	measurer := hirl.Measurer{
		Config: hirl.Config{},
		Methods: []hirl.Method{
			FakeMethodSuccessful{},
			FakeMethodFailure{},
			FakeMethodSuccessful{},
		},
	}
	ctx := context.Background()
	sess := &mockable.Session{
		MockableTestHelpers: map[string][]model.OOAPIService{
			"tcp-echo": {{
				Address: "127.0.0.1",
				Type:    "legacy",
			}},
		},
	}
	measurement := new(model.Measurement)
	callbacks := model.NewPrinterCallbacks(log.Log)
	args := &model.ExperimentArgs{
		Callbacks:   callbacks,
		Measurement: measurement,
		Session:     sess,
	}
	err := measurer.Run(ctx, args)
	if err != nil {
		t.Fatal(err)
	}
	tk := measurement.TestKeys.(*hirl.TestKeys)
	if len(tk.FailureList) != len(tk.Received) {
		t.Fatal("FailureList and Received have different lengths")
	}
	if len(tk.Received) != len(tk.Sent) {
		t.Fatal("Received and Sent have different lengths")
	}
	if len(tk.Sent) != len(tk.TamperingList) {
		t.Fatal("Sent and TamperingList have different lengths")
	}
	for _, failure := range tk.FailureList {
		if failure != nil {
			t.Fatal(*failure)
		}
	}
	for _, received := range tk.Received {
		if received.Value != "antani" {
			t.Fatal("unexpected received value")
		}
	}
	for _, sent := range tk.Sent {
		if sent != "antani" && sent != "melandri" {
			t.Fatal("unexpected sent value")
		}
	}
	var falses, trues int
	for _, entry := range tk.TamperingList {
		if entry {
			trues++
		} else {
			falses++
		}
	}
	if falses != 2 && trues != 1 {
		t.Fatal("not the right values in tampering list")
	}
	if tk.Tampering != true {
		t.Fatal("overall there is no tampering?!")
	}
}

func TestWithNoMethods(t *testing.T) {
	measurer := hirl.Measurer{
		Config:  hirl.Config{},
		Methods: []hirl.Method{},
	}
	ctx := context.Background()
	sess := &mockable.Session{
		MockableTestHelpers: map[string][]model.OOAPIService{
			"tcp-echo": {{
				Address: "127.0.0.1",
				Type:    "legacy",
			}},
		},
	}
	measurement := new(model.Measurement)
	callbacks := model.NewPrinterCallbacks(log.Log)
	args := &model.ExperimentArgs{
		Callbacks:   callbacks,
		Measurement: measurement,
		Session:     sess,
	}
	err := measurer.Run(ctx, args)
	if !errors.Is(err, hirl.ErrNoMeasurementMethod) {
		t.Fatal("not the error we expected")
	}
	tk := measurement.TestKeys.(*hirl.TestKeys)
	if len(tk.FailureList) != 0 {
		t.Fatal("unexpected FailureList length")
	}
	if len(tk.Received) != 0 {
		t.Fatal("unexpected Received length")
	}
	if len(tk.Sent) != 0 {
		t.Fatal("unexpected Sent length")
	}
	if len(tk.TamperingList) != 0 {
		t.Fatal("unexpected TamperingList length")
	}
	if tk.Tampering != false {
		t.Fatal("overall there is tampering?!")
	}
}

func TestNoHelpers(t *testing.T) {
	measurer := hirl.NewExperimentMeasurer(hirl.Config{})
	ctx := context.Background()
	sess := &mockable.Session{}
	measurement := new(model.Measurement)
	callbacks := model.NewPrinterCallbacks(log.Log)
	args := &model.ExperimentArgs{
		Callbacks:   callbacks,
		Measurement: measurement,
		Session:     sess,
	}
	err := measurer.Run(ctx, args)
	if !errors.Is(err, hirl.ErrNoAvailableTestHelpers) {
		t.Fatal("not the error we expected")
	}
	tk := measurement.TestKeys.(*hirl.TestKeys)
	if len(tk.FailureList) != 0 {
		t.Fatal("expected an empty FailureList")
	}
	if len(tk.FailureList) != len(tk.Received) {
		t.Fatal("FailureList and Received have different lengths")
	}
	if len(tk.Received) != len(tk.Sent) {
		t.Fatal("Received and Sent have different lengths")
	}
	if len(tk.Sent) != len(tk.TamperingList) {
		t.Fatal("Sent and TamperingList have different lengths")
	}
	if tk.Tampering != false {
		t.Fatal("overall there is tampering?!")
	}
}

func TestNoActualHelperInList(t *testing.T) {
	measurer := hirl.NewExperimentMeasurer(hirl.Config{})
	ctx := context.Background()
	sess := &mockable.Session{
		MockableTestHelpers: map[string][]model.OOAPIService{
			"tcp-echo": nil,
		},
	}
	measurement := new(model.Measurement)
	callbacks := model.NewPrinterCallbacks(log.Log)
	args := &model.ExperimentArgs{
		Callbacks:   callbacks,
		Measurement: measurement,
		Session:     sess,
	}
	err := measurer.Run(ctx, args)
	if !errors.Is(err, hirl.ErrNoAvailableTestHelpers) {
		t.Fatal("not the error we expected")
	}
	tk := measurement.TestKeys.(*hirl.TestKeys)
	if len(tk.FailureList) != 0 {
		t.Fatal("expected an empty FailureList")
	}
	if len(tk.FailureList) != len(tk.Received) {
		t.Fatal("FailureList and Received have different lengths")
	}
	if len(tk.Received) != len(tk.Sent) {
		t.Fatal("Received and Sent have different lengths")
	}
	if len(tk.Sent) != len(tk.TamperingList) {
		t.Fatal("Sent and TamperingList have different lengths")
	}
	if tk.Tampering != false {
		t.Fatal("overall there is tampering?!")
	}
}

func TestWrongTestHelperType(t *testing.T) {
	measurer := hirl.NewExperimentMeasurer(hirl.Config{})
	ctx := context.Background()
	sess := &mockable.Session{
		MockableTestHelpers: map[string][]model.OOAPIService{
			"tcp-echo": {{
				Address: "127.0.0.1",
				Type:    "antani",
			}},
		},
	}
	measurement := new(model.Measurement)
	callbacks := model.NewPrinterCallbacks(log.Log)
	args := &model.ExperimentArgs{
		Callbacks:   callbacks,
		Measurement: measurement,
		Session:     sess,
	}
	err := measurer.Run(ctx, args)
	if !errors.Is(err, hirl.ErrInvalidHelperType) {
		t.Fatal("not the error we expected")
	}
	tk := measurement.TestKeys.(*hirl.TestKeys)
	if len(tk.FailureList) != 0 {
		t.Fatal("expected an empty FailureList")
	}
	if len(tk.FailureList) != len(tk.Received) {
		t.Fatal("FailureList and Received have different lengths")
	}
	if len(tk.Received) != len(tk.Sent) {
		t.Fatal("Received and Sent have different lengths")
	}
	if len(tk.Sent) != len(tk.TamperingList) {
		t.Fatal("Sent and TamperingList have different lengths")
	}
	if tk.Tampering != false {
		t.Fatal("overall there is tampering?!")
	}
}

func TestRunMethodDialFailure(t *testing.T) {
	sess := &mockable.Session{
		MockableLogger: log.Log,
		MockableTestHelpers: map[string][]model.OOAPIService{
			"tcp-echo": {{
				Address: "37.218.241.93",
				Type:    "legacy",
			}},
		},
	}
	helpers, ok := sess.GetTestHelpersByName("tcp-echo")
	if len(helpers) < 1 || !ok {
		t.Fatal("cannot get helper")
	}
	expected := errors.New("mocked error")
	out := make(chan hirl.MethodResult)
	config := hirl.RunMethodConfig{
		MethodConfig: hirl.MethodConfig{
			Address: helpers[0].Address,
			Logger:  log.Log,
			Out:     out,
		},
		Name: "random_invalid_version_number",
		NewDialer: func(config netx.Config) model.Dialer {
			return FakeDialer{Err: expected}
		},
		RequestLine: "GET / HTTP/ABC",
	}
	go hirl.RunMethod(context.Background(), config)
	result := <-out
	if !errors.Is(result.Err, expected) {
		t.Fatal("not the error we expected")
	}
	if result.Name != "random_invalid_version_number" {
		t.Fatal("unexpected Name")
	}
	if result.Received.Value != "" {
		t.Fatal("unexpected Received.Value")
	}
	if result.Sent != "" {
		t.Fatal("unexpected Sent")
	}
	if result.Tampering != false {
		t.Fatal("unexpected Tampering")
	}
}

func TestRunMethodSetDeadlineFailure(t *testing.T) {
	sess := &mockable.Session{
		MockableLogger: log.Log,
		MockableTestHelpers: map[string][]model.OOAPIService{
			"tcp-echo": {{
				Address: "37.218.241.93",
				Type:    "legacy",
			}},
		},
	}
	helpers, ok := sess.GetTestHelpersByName("tcp-echo")
	if len(helpers) < 1 || !ok {
		t.Fatal("cannot get helper")
	}
	expected := errors.New("mocked error")
	out := make(chan hirl.MethodResult)
	config := hirl.RunMethodConfig{
		MethodConfig: hirl.MethodConfig{
			Address: helpers[0].Address,
			Logger:  log.Log,
			Out:     out,
		},
		Name: "random_invalid_version_number",
		NewDialer: func(config netx.Config) model.Dialer {
			return FakeDialer{Conn: &FakeConn{
				SetDeadlineError: expected,
			}}
		},
		RequestLine: "GET / HTTP/ABC",
	}
	go hirl.RunMethod(context.Background(), config)
	result := <-out
	if !errors.Is(result.Err, expected) {
		t.Fatal("not the error we expected")
	}
	if result.Name != "random_invalid_version_number" {
		t.Fatal("unexpected Name")
	}
	if result.Received.Value != "" {
		t.Fatal("unexpected Received.Value")
	}
	if result.Sent != "" {
		t.Fatal("unexpected Sent")
	}
	if result.Tampering != false {
		t.Fatal("unexpected Tampering")
	}
}

func TestRunMethodWriteFailure(t *testing.T) {
	sess := &mockable.Session{
		MockableLogger: log.Log,
		MockableTestHelpers: map[string][]model.OOAPIService{
			"tcp-echo": {{
				Address: "37.218.241.93",
				Type:    "legacy",
			}},
		},
	}
	helpers, ok := sess.GetTestHelpersByName("tcp-echo")
	if len(helpers) < 1 || !ok {
		t.Fatal("cannot get helper")
	}
	expected := errors.New("mocked error")
	out := make(chan hirl.MethodResult)
	config := hirl.RunMethodConfig{
		MethodConfig: hirl.MethodConfig{
			Address: helpers[0].Address,
			Logger:  log.Log,
			Out:     out,
		},
		Name: "random_invalid_version_number",
		NewDialer: func(config netx.Config) model.Dialer {
			return FakeDialer{Conn: &FakeConn{
				WriteError: expected,
			}}
		},
		RequestLine: "GET / HTTP/ABC",
	}
	go hirl.RunMethod(context.Background(), config)
	result := <-out
	if !errors.Is(result.Err, expected) {
		t.Fatal("not the error we expected")
	}
	if result.Name != "random_invalid_version_number" {
		t.Fatal("unexpected Name")
	}
	if result.Received.Value != "" {
		t.Fatal("unexpected Received.Value")
	}
	if result.Sent != "" {
		t.Fatal("unexpected Sent")
	}
	if result.Tampering != false {
		t.Fatal("unexpected Tampering")
	}
}

func TestRunMethodReadEOFWithWrongData(t *testing.T) {
	sess := &mockable.Session{
		MockableLogger: log.Log,
		MockableTestHelpers: map[string][]model.OOAPIService{
			"tcp-echo": {{
				Address: "37.218.241.93",
				Type:    "legacy",
			}},
		},
	}
	helpers, ok := sess.GetTestHelpersByName("tcp-echo")
	if len(helpers) < 1 || !ok {
		t.Fatal("cannot get helper")
	}
	out := make(chan hirl.MethodResult)
	config := hirl.RunMethodConfig{
		MethodConfig: hirl.MethodConfig{
			Address: helpers[0].Address,
			Logger:  log.Log,
			Out:     out,
		},
		Name: "random_invalid_version_number",
		NewDialer: func(config netx.Config) model.Dialer {
			return FakeDialer{Conn: &FakeConn{
				ReadData: []byte("0xdeadbeef"),
			}}
		},
		RequestLine: "GET / HTTP/ABC",
	}
	go hirl.RunMethod(context.Background(), config)
	result := <-out
	if !errors.Is(result.Err, io.EOF) {
		t.Fatal("not the error we expected")
	}
	if result.Name != "random_invalid_version_number" {
		t.Fatal("unexpected Name")
	}
	if result.Received.Value != "0xdeadbeef" {
		t.Fatal("unexpected Received.Value")
	}
	if result.Sent != "GET / HTTP/ABC" {
		t.Fatal("unexpected Sent")
	}
	if result.Tampering != true {
		t.Fatal("unexpected Tampering")
	}
}

func TestSummaryKeysInvalidType(t *testing.T) {
	measurement := new(model.Measurement)
	m := &hirl.Measurer{}
	_, err := m.GetSummaryKeys(measurement)
	if err.Error() != "invalid test keys type" {
		t.Fatal("not the error we expected")
	}
}

func TestSummaryKeysFalse(t *testing.T) {
	measurement := &model.Measurement{TestKeys: &hirl.TestKeys{
		Tampering: false,
	}}
	m := &hirl.Measurer{}
	osk, err := m.GetSummaryKeys(measurement)
	if err != nil {
		t.Fatal(err)
	}
	sk := osk.(hirl.SummaryKeys)
	if sk.IsAnomaly {
		t.Fatal("invalid isAnomaly")
	}
}

func TestSummaryKeysTrue(t *testing.T) {
	measurement := &model.Measurement{TestKeys: &hirl.TestKeys{
		Tampering: true,
	}}
	m := &hirl.Measurer{}
	osk, err := m.GetSummaryKeys(measurement)
	if err != nil {
		t.Fatal(err)
	}
	sk := osk.(hirl.SummaryKeys)
	if sk.IsAnomaly == false {
		t.Fatal("invalid isAnomaly")
	}
}
