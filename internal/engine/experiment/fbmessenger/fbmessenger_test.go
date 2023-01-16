package fbmessenger_test

import (
	"context"
	"io"
	"testing"

	"github.com/apex/log"
	engine "github.com/ooni/probe-cli/v3/internal/engine"
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/fbmessenger"
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/urlgetter"
	"github.com/ooni/probe-cli/v3/internal/legacy/mockable"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/tracex"
)

func TestNewExperimentMeasurer(t *testing.T) {
	measurer := fbmessenger.NewExperimentMeasurer(fbmessenger.Config{})
	if measurer.ExperimentName() != "facebook_messenger" {
		t.Fatal("unexpected name")
	}
	if measurer.ExperimentVersion() != "0.2.0" {
		t.Fatal("unexpected version")
	}
}

func TestSuccess(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	measurer := fbmessenger.NewExperimentMeasurer(fbmessenger.Config{})
	ctx := context.Background()
	// we need a real session because we need the ASN database
	sess := newsession(t)
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
	tk := measurement.TestKeys.(*fbmessenger.TestKeys)
	if *tk.FacebookBAPIDNSConsistent != true {
		t.Fatal("invalid FacebookBAPIDNSConsistent")
	}
	if *tk.FacebookBAPIReachable != true {
		t.Fatal("invalid FacebookBAPIReachable")
	}
	if *tk.FacebookBGraphDNSConsistent != true {
		t.Fatal("invalid FacebookBGraphDNSConsistent")
	}
	if *tk.FacebookBGraphReachable != true {
		t.Fatal("invalid FacebookBGraphReachable")
	}
	if *tk.FacebookEdgeDNSConsistent != true {
		t.Fatal("invalid FacebookEdgeDNSConsistent")
	}
	if *tk.FacebookEdgeReachable != true {
		t.Fatal("invalid FacebookEdgeReachable")
	}
	if *tk.FacebookExternalCDNDNSConsistent != true {
		t.Fatal("invalid FacebookExternalCDNDNSConsistent")
	}
	if *tk.FacebookExternalCDNReachable != true {
		t.Fatal("invalid FacebookExternalCDNReachable")
	}
	if *tk.FacebookScontentCDNDNSConsistent != true {
		t.Fatal("invalid FacebookScontentCDNDNSConsistent")
	}
	if *tk.FacebookScontentCDNReachable != true {
		t.Fatal("invalid FacebookScontentCDNReachable")
	}
	if *tk.FacebookStarDNSConsistent != true {
		t.Fatal("invalid FacebookStarDNSConsistent")
	}
	if *tk.FacebookStarReachable != true {
		t.Fatal("invalid FacebookStarReachable")
	}
	if *tk.FacebookSTUNDNSConsistent != true {
		t.Fatal("invalid FacebookSTUNDNSConsistent")
	}
	if tk.FacebookSTUNReachable != nil {
		t.Fatal("invalid FacebookSTUNReachable")
	}
	if *tk.FacebookDNSBlocking != false {
		t.Fatal("invalid FacebookDNSBlocking")
	}
	if *tk.FacebookTCPBlocking != false {
		t.Fatal("invalid FacebookTCPBlocking")
	}
}

func TestWithCancelledContext(t *testing.T) {
	measurer := fbmessenger.NewExperimentMeasurer(fbmessenger.Config{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // so we fail immediately
	sess := &mockable.Session{MockableLogger: log.Log}
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
	tk := measurement.TestKeys.(*fbmessenger.TestKeys)
	if *tk.FacebookBAPIDNSConsistent != false {
		t.Fatal("invalid FacebookBAPIDNSConsistent")
	}
	if tk.FacebookBAPIReachable != nil {
		t.Fatal("invalid FacebookBAPIReachable")
	}
	if *tk.FacebookBGraphDNSConsistent != false {
		t.Fatal("invalid FacebookBGraphDNSConsistent")
	}
	if tk.FacebookBGraphReachable != nil {
		t.Fatal("invalid FacebookBGraphReachable")
	}
	if *tk.FacebookEdgeDNSConsistent != false {
		t.Fatal("invalid FacebookEdgeDNSConsistent")
	}
	if tk.FacebookEdgeReachable != nil {
		t.Fatal("invalid FacebookEdgeReachable")
	}
	if *tk.FacebookExternalCDNDNSConsistent != false {
		t.Fatal("invalid FacebookExternalCDNDNSConsistent")
	}
	if tk.FacebookExternalCDNReachable != nil {
		t.Fatal("invalid FacebookExternalCDNReachable")
	}
	if *tk.FacebookScontentCDNDNSConsistent != false {
		t.Fatal("invalid FacebookScontentCDNDNSConsistent")
	}
	if tk.FacebookScontentCDNReachable != nil {
		t.Fatal("invalid FacebookScontentCDNReachable")
	}
	if *tk.FacebookStarDNSConsistent != false {
		t.Fatal("invalid FacebookStarDNSConsistent")
	}
	if tk.FacebookStarReachable != nil {
		t.Fatal("invalid FacebookStarReachable")
	}
	if *tk.FacebookSTUNDNSConsistent != false {
		t.Fatal("invalid FacebookSTUNDNSConsistent")
	}
	if tk.FacebookSTUNReachable != nil {
		t.Fatal("invalid FacebookSTUNReachable")
	}
	if *tk.FacebookDNSBlocking != true {
		t.Fatal("invalid FacebookDNSBlocking")
	}
	// no TCP blocking because we didn't ever reach TCP connect
	if *tk.FacebookTCPBlocking != false {
		t.Fatal("invalid FacebookTCPBlocking")
	}
	sk, err := measurer.GetSummaryKeys(measurement)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := sk.(fbmessenger.SummaryKeys); !ok {
		t.Fatal("invalid type for summary keys")
	}
}

func TestComputeEndpointStatsTCPBlocking(t *testing.T) {
	failure := io.EOF.Error()
	operation := netxlite.ConnectOperation
	tk := fbmessenger.TestKeys{}
	tk.Update(urlgetter.MultiOutput{
		Input: urlgetter.MultiInput{Target: fbmessenger.ServiceEdge},
		TestKeys: urlgetter.TestKeys{
			Failure:         &failure,
			FailedOperation: &operation,
			Queries: []tracex.DNSQueryEntry{{
				Answers: []tracex.DNSAnswerEntry{{
					ASN: fbmessenger.FacebookASN,
				}},
			}},
		},
	})
	if *tk.FacebookEdgeDNSConsistent != true {
		t.Fatal("invalid FacebookEdgeDNSConsistent")
	}
	if *tk.FacebookEdgeReachable != false {
		t.Fatal("invalid FacebookEdgeReachable")
	}
	if tk.FacebookDNSBlocking != nil { // meaning: not determined yet
		t.Fatal("invalid FacebookDNSBlocking")
	}
	if *tk.FacebookTCPBlocking != true {
		t.Fatal("invalid FacebookTCPBlocking")
	}
}

func TestComputeEndpointStatsDNSIsLying(t *testing.T) {
	failure := io.EOF.Error()
	operation := netxlite.ConnectOperation
	tk := fbmessenger.TestKeys{}
	tk.Update(urlgetter.MultiOutput{
		Input: urlgetter.MultiInput{Target: fbmessenger.ServiceEdge},
		TestKeys: urlgetter.TestKeys{
			Failure:         &failure,
			FailedOperation: &operation,
			Queries: []tracex.DNSQueryEntry{{
				Answers: []tracex.DNSAnswerEntry{{
					ASN: 0,
				}},
			}},
		},
	})
	if *tk.FacebookEdgeDNSConsistent != false {
		t.Fatal("invalid FacebookEdgeDNSConsistent")
	}
	if tk.FacebookEdgeReachable != nil {
		t.Fatal("invalid FacebookEdgeReachable")
	}
	if *tk.FacebookDNSBlocking != true {
		t.Fatal("invalid FacebookDNSBlocking")
	}
	if tk.FacebookTCPBlocking != nil { // meaning: not determined yet
		t.Fatal("invalid FacebookTCPBlocking")
	}
}

func newsession(t *testing.T) model.ExperimentSession {
	sess, err := engine.NewSession(context.Background(), engine.SessionConfig{
		AvailableProbeServices: []model.OOAPIService{{
			Address: "https://ams-pg-test.ooni.org",
			Type:    "https",
		}},
		Logger:          log.Log,
		SoftwareName:    "ooniprobe-engine",
		SoftwareVersion: "0.0.1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := sess.MaybeLookupLocation(); err != nil {
		t.Fatal(err)
	}
	return sess
}

func TestSummaryKeysInvalidType(t *testing.T) {
	measurement := new(model.Measurement)
	m := &fbmessenger.Measurer{}
	_, err := m.GetSummaryKeys(measurement)
	if err.Error() != "invalid test keys type" {
		t.Fatal("not the error we expected")
	}
}

func TestSummaryKeysWithNils(t *testing.T) {
	measurement := &model.Measurement{TestKeys: &fbmessenger.TestKeys{}}
	m := &fbmessenger.Measurer{}
	osk, err := m.GetSummaryKeys(measurement)
	if err != nil {
		t.Fatal(err)
	}
	sk := osk.(fbmessenger.SummaryKeys)
	if sk.DNSBlocking {
		t.Fatal("invalid dnsBlocking")
	}
	if sk.TCPBlocking {
		t.Fatal("invalid tcpBlocking")
	}
	if sk.IsAnomaly {
		t.Fatal("invalid isAnomaly")
	}
}

func TestSummaryKeysWithFalseFalse(t *testing.T) {
	falsy := false
	measurement := &model.Measurement{TestKeys: &fbmessenger.TestKeys{
		FacebookTCPBlocking: &falsy,
		FacebookDNSBlocking: &falsy,
	}}
	m := &fbmessenger.Measurer{}
	osk, err := m.GetSummaryKeys(measurement)
	if err != nil {
		t.Fatal(err)
	}
	sk := osk.(fbmessenger.SummaryKeys)
	if sk.DNSBlocking {
		t.Fatal("invalid dnsBlocking")
	}
	if sk.TCPBlocking {
		t.Fatal("invalid tcpBlocking")
	}
	if sk.IsAnomaly {
		t.Fatal("invalid isAnomaly")
	}
}

func TestSummaryKeysWithFalseTrue(t *testing.T) {
	falsy := false
	truy := true
	measurement := &model.Measurement{TestKeys: &fbmessenger.TestKeys{
		FacebookTCPBlocking: &falsy,
		FacebookDNSBlocking: &truy,
	}}
	m := &fbmessenger.Measurer{}
	osk, err := m.GetSummaryKeys(measurement)
	if err != nil {
		t.Fatal(err)
	}
	sk := osk.(fbmessenger.SummaryKeys)
	if sk.DNSBlocking == false {
		t.Fatal("invalid dnsBlocking")
	}
	if sk.TCPBlocking {
		t.Fatal("invalid tcpBlocking")
	}
	if sk.IsAnomaly == false {
		t.Fatal("invalid isAnomaly")
	}
}

func TestSummaryKeysWithTrueFalse(t *testing.T) {
	falsy := false
	truy := true
	measurement := &model.Measurement{TestKeys: &fbmessenger.TestKeys{
		FacebookTCPBlocking: &truy,
		FacebookDNSBlocking: &falsy,
	}}
	m := &fbmessenger.Measurer{}
	osk, err := m.GetSummaryKeys(measurement)
	if err != nil {
		t.Fatal(err)
	}
	sk := osk.(fbmessenger.SummaryKeys)
	if sk.DNSBlocking {
		t.Fatal("invalid dnsBlocking")
	}
	if sk.TCPBlocking == false {
		t.Fatal("invalid tcpBlocking")
	}
	if sk.IsAnomaly == false {
		t.Fatal("invalid isAnomaly")
	}
}

func TestSummaryKeysWithTrueTrue(t *testing.T) {
	truy := true
	measurement := &model.Measurement{TestKeys: &fbmessenger.TestKeys{
		FacebookTCPBlocking: &truy,
		FacebookDNSBlocking: &truy,
	}}
	m := &fbmessenger.Measurer{}
	osk, err := m.GetSummaryKeys(measurement)
	if err != nil {
		t.Fatal(err)
	}
	sk := osk.(fbmessenger.SummaryKeys)
	if sk.DNSBlocking == false {
		t.Fatal("invalid dnsBlocking")
	}
	if sk.TCPBlocking == false {
		t.Fatal("invalid tcpBlocking")
	}
	if sk.IsAnomaly == false {
		t.Fatal("invalid isAnomaly")
	}
}
