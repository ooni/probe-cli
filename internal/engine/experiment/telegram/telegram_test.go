package telegram_test

import (
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/engine/experiment/telegram"
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/urlgetter"
	"github.com/ooni/probe-cli/v3/internal/legacy/mockable"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func TestNewExperimentMeasurer(t *testing.T) {
	measurer := telegram.NewExperimentMeasurer(telegram.Config{})
	if measurer.ExperimentName() != "telegram" {
		t.Fatal("unexpected name")
	}
	if measurer.ExperimentVersion() != "0.3.0" {
		t.Fatal("unexpected version")
	}
}

func TestGood(t *testing.T) {
	measurer := telegram.NewExperimentMeasurer(telegram.Config{})
	measurement := new(model.Measurement)
	args := &model.ExperimentArgs{
		Callbacks:   model.NewPrinterCallbacks(model.DiscardLogger),
		Measurement: measurement,
		Session: &mockable.Session{
			MockableLogger: model.DiscardLogger,
		},
	}
	err := measurer.Run(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	tk := measurement.TestKeys.(*telegram.TestKeys)
	if tk.Agent != "redirect" {
		t.Fatal("unexpected Agent")
	}
	if tk.FailedOperation != nil {
		t.Fatal("unexpected FailedOperation")
	}
	if tk.Failure != nil {
		t.Fatal("unexpected Failure")
	}
	if len(tk.NetworkEvents) <= 0 {
		t.Fatal("no NetworkEvents?!")
	}
	if len(tk.Queries) <= 0 {
		t.Fatal("no Queries?!")
	}
	if len(tk.Requests) <= 0 {
		t.Fatal("no Requests?!")
	}
	if len(tk.TCPConnect) <= 0 {
		t.Fatal("no TCPConnect?!")
	}
	if len(tk.TLSHandshakes) <= 0 {
		t.Fatal("no TLSHandshakes?!")
	}
	if tk.TelegramHTTPBlocking != false {
		t.Fatal("unexpected TelegramHTTPBlocking")
	}
	if tk.TelegramTCPBlocking != false {
		t.Fatal("unexpected TelegramTCPBlocking")
	}
	if tk.TelegramWebFailure != nil {
		t.Fatal("unexpected TelegramWebFailure")
	}
	if tk.TelegramWebStatus != "ok" {
		t.Fatal("unexpected TelegramWebStatus")
	}
	sk, err := measurer.GetSummaryKeys(measurement)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := sk.(telegram.SummaryKeys); !ok {
		t.Fatal("invalid type for summary keys")
	}
}

func TestUpdateWithNoAccessPointsBlocking(t *testing.T) {
	tk := telegram.NewTestKeys()
	tk.Update(urlgetter.MultiOutput{
		Input: urlgetter.MultiInput{
			Config: urlgetter.Config{Method: "POST"},
			Target: "http://149.154.175.50/",
		},
		TestKeys: urlgetter.TestKeys{
			Failure: (func() *string {
				s := netxlite.FailureEOFError
				return &s
			})(),
		},
	})
	tk.Update(urlgetter.MultiOutput{
		Input: urlgetter.MultiInput{
			Config: urlgetter.Config{Method: "POST"},
			Target: "http://149.154.175.50:443/",
		},
		TestKeys: urlgetter.TestKeys{
			Failure: nil, // this should be enough to declare success
		},
	})
	if tk.TelegramHTTPBlocking == true {
		t.Fatal("there should be no TelegramHTTPBlocking")
	}
	if tk.TelegramTCPBlocking == true {
		t.Fatal("there should be no TelegramTCPBlocking")
	}
}

func TestUpdateWithNilFailedOperation(t *testing.T) {
	tk := telegram.NewTestKeys()
	tk.Update(urlgetter.MultiOutput{
		Input: urlgetter.MultiInput{
			Config: urlgetter.Config{Method: "POST"},
			Target: "http://149.154.175.50/",
		},
		TestKeys: urlgetter.TestKeys{
			Failure: (func() *string {
				s := netxlite.FailureEOFError
				return &s
			})(),
		},
	})
	tk.Update(urlgetter.MultiOutput{
		Input: urlgetter.MultiInput{
			Config: urlgetter.Config{Method: "POST"},
			Target: "http://149.154.175.50:443/",
		},
		TestKeys: urlgetter.TestKeys{
			Failure: (func() *string {
				s := netxlite.FailureEOFError
				return &s
			})(),
		},
	})
	if tk.TelegramHTTPBlocking == false {
		t.Fatal("there should be TelegramHTTPBlocking")
	}
	if tk.TelegramTCPBlocking == true {
		t.Fatal("there should be no TelegramTCPBlocking")
	}
}

func TestUpdateWithNonConnectFailedOperation(t *testing.T) {
	tk := telegram.NewTestKeys()
	tk.Update(urlgetter.MultiOutput{
		Input: urlgetter.MultiInput{
			Config: urlgetter.Config{Method: "POST"},
			Target: "http://149.154.175.50/",
		},
		TestKeys: urlgetter.TestKeys{
			FailedOperation: (func() *string {
				s := netxlite.ConnectOperation
				return &s
			})(),
			Failure: (func() *string {
				s := netxlite.FailureEOFError
				return &s
			})(),
		},
	})
	tk.Update(urlgetter.MultiOutput{
		Input: urlgetter.MultiInput{
			Config: urlgetter.Config{Method: "POST"},
			Target: "http://149.154.175.50:443/",
		},
		TestKeys: urlgetter.TestKeys{
			FailedOperation: (func() *string {
				s := netxlite.HTTPRoundTripOperation
				return &s
			})(),
			Failure: (func() *string {
				s := netxlite.FailureEOFError
				return &s
			})(),
		},
	})
	if tk.TelegramHTTPBlocking == false {
		t.Fatal("there should be TelegramHTTPBlocking")
	}
	if tk.TelegramTCPBlocking == true {
		t.Fatal("there should be no TelegramTCPBlocking")
	}
}

func TestUpdateWithAllConnectsFailed(t *testing.T) {
	tk := telegram.NewTestKeys()
	tk.Update(urlgetter.MultiOutput{
		Input: urlgetter.MultiInput{
			Config: urlgetter.Config{Method: "POST"},
			Target: "http://149.154.175.50/",
		},
		TestKeys: urlgetter.TestKeys{
			FailedOperation: (func() *string {
				s := netxlite.ConnectOperation
				return &s
			})(),
			Failure: (func() *string {
				s := netxlite.FailureEOFError
				return &s
			})(),
		},
	})
	tk.Update(urlgetter.MultiOutput{
		Input: urlgetter.MultiInput{
			Config: urlgetter.Config{Method: "POST"},
			Target: "http://149.154.175.50:443/",
		},
		TestKeys: urlgetter.TestKeys{
			FailedOperation: (func() *string {
				s := netxlite.ConnectOperation
				return &s
			})(),
			Failure: (func() *string {
				s := netxlite.FailureEOFError
				return &s
			})(),
		},
	})
	if tk.TelegramHTTPBlocking == false {
		t.Fatal("there should be TelegramHTTPBlocking")
	}
	if tk.TelegramTCPBlocking == false {
		t.Fatal("there should be TelegramTCPBlocking")
	}
}

func TestUpdateWithWebFailure(t *testing.T) {
	failure := netxlite.FailureEOFError
	failedOperation := netxlite.TLSHandshakeOperation
	tk := telegram.NewTestKeys()
	tk.Update(urlgetter.MultiOutput{
		Input: urlgetter.MultiInput{
			Config: urlgetter.Config{Method: "GET"},
			Target: "https://web.telegram.org/",
		},
		TestKeys: urlgetter.TestKeys{
			Failure:         &failure,
			FailedOperation: &failedOperation,
		},
	})
	if tk.TelegramWebStatus != "blocked" {
		t.Fatal("TelegramWebStatus should be blocked")
	}
	if *tk.TelegramWebFailure != failure {
		t.Fatal("invalid TelegramWebFailure")
	}
}

func TestUpdateWithAllGood(t *testing.T) {
	tk := telegram.NewTestKeys()
	tk.Update(urlgetter.MultiOutput{
		Input: urlgetter.MultiInput{
			Config: urlgetter.Config{Method: "GET"},
			Target: "https://web.telegram.org/",
		},
		TestKeys: urlgetter.TestKeys{
			HTTPResponseStatus: 200,
			HTTPResponseBody:   "<HTML><title>Telegram Web</title></HTML>",
		},
	})
	if tk.TelegramWebStatus != "ok" {
		t.Fatal("TelegramWebStatus should be ok")
	}
	if tk.TelegramWebFailure != nil {
		t.Fatal("invalid TelegramWebFailure")
	}
}

func TestSummaryKeysInvalidType(t *testing.T) {
	measurement := new(model.Measurement)
	m := &telegram.Measurer{}
	_, err := m.GetSummaryKeys(measurement)
	if err.Error() != "invalid test keys type" {
		t.Fatal("not the error we expected")
	}
}

func TestSummaryKeysWorksAsIntended(t *testing.T) {
	failure := io.EOF.Error()
	tests := []struct {
		tk        telegram.TestKeys
		isAnomaly bool
	}{{
		tk:        telegram.TestKeys{},
		isAnomaly: false,
	}, {
		tk:        telegram.TestKeys{TelegramTCPBlocking: true},
		isAnomaly: true,
	}, {
		tk:        telegram.TestKeys{TelegramHTTPBlocking: true},
		isAnomaly: true,
	}, {
		tk:        telegram.TestKeys{TelegramWebFailure: &failure},
		isAnomaly: true,
	}}
	for idx, tt := range tests {
		t.Run(fmt.Sprintf("%d", idx), func(t *testing.T) {
			m := &telegram.Measurer{}
			measurement := &model.Measurement{TestKeys: &tt.tk}
			got, err := m.GetSummaryKeys(measurement)
			if err != nil {
				t.Fatal(err)
				return
			}
			sk := got.(telegram.SummaryKeys)
			if sk.IsAnomaly != tt.isAnomaly {
				t.Fatal("unexpected isAnomaly value")
			}
		})
	}
}
