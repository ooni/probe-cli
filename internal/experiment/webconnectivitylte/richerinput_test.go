package webconnectivitylte

import (
	"context"
	"errors"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/targetloading"
)

func TestTarget(t *testing.T) {
	t.Run("methods preserve category and country", func(t *testing.T) {
		target := &Target{
			URL:      "https://www.example.com/",
			Config:   &Config{},
			category: "NEWS",
			country:  "IT",
		}
		if target.Category() != "NEWS" {
			t.Fatal("invalid Category")
		}
		if target.Country() != "IT" {
			t.Fatal("invalid Country")
		}
		if target.Input() != "https://www.example.com/" {
			t.Fatal("invalid Input")
		}
		if target.String() != "https://www.example.com/" {
			t.Fatal("invalid String")
		}
		if len(target.Options()) != 0 {
			t.Fatal("expected no options for an empty Config")
		}
	})

	t.Run("empty category and country fall back to defaults", func(t *testing.T) {
		target := &Target{URL: "https://x/", Config: &Config{}}
		if target.Category() != model.DefaultCategoryCode {
			t.Fatal("invalid default Category")
		}
		if target.Country() != model.DefaultCountryCode {
			t.Fatal("invalid default Country")
		}
	})
}

func TestNewLoader(t *testing.T) {
	child := &targetloading.Loader{}
	options := &Config{}
	loader := NewLoader(child, options).(*targetLoader)
	if child != loader.loader {
		t.Fatal("invalid loader pointer")
	}
	if options != loader.options {
		t.Fatal("invalid options pointer")
	}
}

func TestTargetLoaderLoad(t *testing.T) {
	t.Run("wraps static inputs into richer-input targets", func(t *testing.T) {
		tl := &targetLoader{
			options: &Config{},
			loader: &targetloading.Loader{
				ExperimentName: "web_connectivity",
				InputPolicy:    model.InputOrQueryBackend,
				Logger:         model.DiscardLogger,
				Session:        &mocks.Session{},
				StaticInputs:   []string{"https://www.example.com/"},
			},
		}
		targets, err := tl.Load(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if len(targets) != 1 {
			t.Fatal("expected exactly one target")
		}
		got, ok := targets[0].(*Target)
		if !ok {
			t.Fatal("expected a *Target")
		}
		if got.Input() != "https://www.example.com/" {
			t.Fatal("unexpected URL")
		}
		// CLI/file inputs carry the default category and country codes.
		if got.Category() != model.DefaultCategoryCode {
			t.Fatal("unexpected category")
		}
		if got.Country() != model.DefaultCountryCode {
			t.Fatal("unexpected country")
		}
	})
}

func TestMeasurerRunWithInvalidTarget(t *testing.T) {
	measurer := NewExperimentMeasurer()

	t.Run("with nil target we get ErrInputRequired", func(t *testing.T) {
		err := measurer.Run(context.Background(), &model.ExperimentArgs{
			Callbacks:   model.NewPrinterCallbacks(model.DiscardLogger),
			Measurement: &model.Measurement{},
			Session:     &mocks.Session{},
		})
		if !errors.Is(err, ErrInputRequired) {
			t.Fatal("unexpected error", err)
		}
	})

	t.Run("with the wrong target type we get ErrInvalidInputType", func(t *testing.T) {
		err := measurer.Run(context.Background(), &model.ExperimentArgs{
			Callbacks:   model.NewPrinterCallbacks(model.DiscardLogger),
			Measurement: &model.Measurement{},
			Session:     &mocks.Session{},
			Target:      &model.OOAPIURLInfo{},
		})
		if !errors.Is(err, ErrInvalidInputType) {
			t.Fatal("unexpected error", err)
		}
	})
}
