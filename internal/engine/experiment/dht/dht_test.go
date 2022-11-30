package dht

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"testing"

	"github.com/anacrolix/dht/v2"
	"github.com/ooni/probe-cli/v3/internal/engine/mockable"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func TestMeasurer_run(t *testing.T) {
	// runHelper is an helper function to run this set of tests.
	runHelper := func(input string) (*model.Measurement, model.ExperimentMeasurer, error) {
		measurer := NewExperimentMeasurer(Config{})
		ctx := context.Background()
		measurement := &model.Measurement{
			Input: model.MeasurementTarget(input),
		}
		session := &mockable.Session{
			MockableLogger: model.DiscardLogger,
		}

		args := &model.ExperimentArgs{
			Callbacks:   model.NewPrinterCallbacks(model.DiscardLogger),
			Measurement: measurement,
			Session:     session,
		}

		err := measurer.Run(ctx, args)
		return measurement, measurer, err
	}

	t.Run("with empty input", func(t *testing.T) {
		_, _, err := runHelper("")
		if !errors.Is(err, errNoInputProvided) {
			t.Fatal("unexpected error", err)
		}
	})

	t.Run("with invalid URL", func(t *testing.T) {
		_, _, err := runHelper("\t")
		if !errors.Is(err, errInputIsNotAnURL) {
			t.Fatal("unexpected error", err)
		}
	})

	t.Run("with invalid scheme", func(t *testing.T) {
		_, _, err := runHelper("https://8.8.8.8:443/")
		if !errors.Is(err, errInvalidScheme) {
			t.Fatal("unexpected error", err)
		}
	})

	t.Run("with missing port", func(t *testing.T) {
		_, _, err := runHelper("dht://8.8.8.8")
		if !errors.Is(err, errMissingPort) {
			t.Fatal("unexpected error", err)
		}
	})

	t.Run("with local listener", func(t *testing.T) {
		conf := new(dht.ServerConfig)
		conf.StartingNodes = func() (addrs []dht.Addr, err error) {
			return []dht.Addr{}, nil
		}
		conf.Passive = false
		dht, err := dht.NewServer(conf)
		if err != nil {
			log.Fatal(err)
		}
		defer dht.Close()
		_, _ = dht.Bootstrap()

		url := fmt.Sprintf("dht://%s", dht.Addr().String())

		hash := "631a31dd0a46257d5078c0dee4e66e26f73e42ac"
		var infohash [20]byte
		copy(infohash[:], hash)
		_, _ = dht.AnnounceTraversal(infohash)

		meas, m, err := runHelper(url)

		tk := meas.TestKeys.(*TestKeys)
		bs, _ := json.MarshalIndent(tk, "", "  ")
		println(string(bs))

		if err != nil {
			t.Fatal(err)
		}

		if tk.Failure != "" {
			t.Fatal(tk.Failure)
		}

		if len(tk.Runs) != 1 {
			t.Fatal("Expected one DHT run")
		}

		run := tk.Runs[0]

		if run.Failure != "" {
			t.Fatal(run.Failure)
		}

		if run.BootstrapNum != 1 {
			t.Fatal("Expected only one bootstrap node")
		}

		if run.PeersRespondedNum != 1 {
			t.Fatal("Expected bootstrap node to respond")
		}

		ask, err := m.GetSummaryKeys(meas)
		if err != nil {
			t.Fatal("cannot obtain summary")
		}
		summary := ask.(SummaryKeys)
		if summary.IsAnomaly {
			t.Fatal("expected no anomaly")
		}
	})
}
