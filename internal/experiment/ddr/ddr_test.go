package ddr

import (
	"context"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func TestMeasurerExperimentNameVersion(t *testing.T) {
	measurer := NewExperimentMeasurer(Config{})
	if measurer.ExperimentName() != "ddr" {
		t.Fatal("unexpected ExperimentName")
	}
	if measurer.ExperimentVersion() != "0.1.0" {
		t.Fatal("unexpected ExperimentVersion")
	}
}

func TestMeasurerRun(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}

	oneOneOneOneResolver := "1.1.1.1:53"

	measurer := NewExperimentMeasurer(Config{
		CustomResolver: &oneOneOneOneResolver,
	})
	args := &model.ExperimentArgs{
		Callbacks:   model.NewPrinterCallbacks(log.Log),
		Measurement: new(model.Measurement),
		Session: &mocks.Session{
			MockLogger: func() model.Logger {
				return log.Log
			},
		},
	}
	if err := measurer.Run(context.Background(), args); err != nil {
		t.Fatal(err)
	}
	tk := args.Measurement.TestKeys.(*TestKeys)
	if tk.Failure != nil {
		t.Fatal("unexpected Failure")
	}

	firstAnswer := tk.Queries.Answers[0]

	if firstAnswer.AnswerType != "SVCB" {
		t.Fatal("unexpected AnswerType")
	}

	if tk.Queries.ResolverAddress != oneOneOneOneResolver {
		t.Fatal("Resolver should be written to TestKeys")
	}

	// 1.1.1.1 supports DDR
	if tk.SupportsDDR != true {
		t.Fatal("unexpected value for Supports DDR")
	}
}

// This test fails because the resolver is a domain name and not an IP address.
func TestMeasurerFailsWithDomainResolver(t *testing.T) {
	invalidResolver := "invalid-resolver.example:53"

	tk, _ := runExperiment(invalidResolver)
	if tk.Failure == nil {
		t.Fatal("expected Failure")
	}
}

func TestMeasurerFailsWithNoPort(t *testing.T) {
	invalidResolver := "1.1.1.1"

	tk, _ := runExperiment(invalidResolver)
	if tk.Failure == nil {
		t.Fatal("expected Failure")
	}
}

func runExperiment(resolver string) (*TestKeys, error) {
	measurer := NewExperimentMeasurer(Config{
		CustomResolver: &resolver,
	})
	args := &model.ExperimentArgs{
		Callbacks:   model.NewPrinterCallbacks(log.Log),
		Measurement: new(model.Measurement),
		Session: &mocks.Session{
			MockLogger: func() model.Logger {
				return log.Log
			},
		},
	}
	err := measurer.Run(context.Background(), args)
	return args.Measurement.TestKeys.(*TestKeys), err
}
