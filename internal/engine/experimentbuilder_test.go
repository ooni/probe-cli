package engine

import (
	"testing"

	"github.com/ooni/probe-cli/v3/internal/engine/experiment/example"
)

func TestExperimentBuilderOptions(t *testing.T) {
	t.Run("when config is not a pointer", func(t *testing.T) {
		b := &ExperimentBuilder{
			config: 17,
		}
		options, err := b.Options()
		if err == nil {
			t.Fatal("expected an error here")
		}
		if options != nil {
			t.Fatal("expected nil here")
		}
	})
	t.Run("when config is not a struct", func(t *testing.T) {
		number := 17
		b := &ExperimentBuilder{
			config: &number,
		}
		options, err := b.Options()
		if err == nil {
			t.Fatal("expected an error here")
		}
		if options != nil {
			t.Fatal("expected nil here")
		}
	})
}

func TestExperimentBuilderSetOption(t *testing.T) {
	t.Run("when config is not a pointer", func(t *testing.T) {
		b := &ExperimentBuilder{
			config: 17,
		}
		if err := b.SetOptionBool("antani", false); err == nil {
			t.Fatal("expected an error here")
		}
	})
	t.Run("when config is not a struct", func(t *testing.T) {
		number := 17
		b := &ExperimentBuilder{
			config: &number,
		}
		if err := b.SetOptionBool("antani", false); err == nil {
			t.Fatal("expected an error here")
		}
	})
	t.Run("when field is not valid", func(t *testing.T) {
		b := &ExperimentBuilder{
			config: &ExperimentBuilder{},
		}
		if err := b.SetOptionBool("antani", false); err == nil {
			t.Fatal("expected an error here")
		}
	})
	t.Run("when field is not bool", func(t *testing.T) {
		b := &ExperimentBuilder{
			config: new(example.Config),
		}
		if err := b.SetOptionBool("Message", false); err == nil {
			t.Fatal("expected an error here")
		}
	})
	t.Run("when field is not string", func(t *testing.T) {
		b := &ExperimentBuilder{
			config: new(example.Config),
		}
		if err := b.SetOptionString("ReturnError", "xx"); err == nil {
			t.Fatal("expected an error here")
		}
	})
	t.Run("when field is not int", func(t *testing.T) {
		b := &ExperimentBuilder{
			config: new(example.Config),
		}
		if err := b.SetOptionInt("ReturnError", 17); err == nil {
			t.Fatal("expected an error here")
		}
	})
	t.Run("when int field does not exist", func(t *testing.T) {
		b := &ExperimentBuilder{
			config: new(example.Config),
		}
		if err := b.SetOptionInt("antani", 17); err == nil {
			t.Fatal("expected an error here")
		}
	})
	t.Run("when string field does not exist", func(t *testing.T) {
		b := &ExperimentBuilder{
			config: new(example.Config),
		}
		if err := b.SetOptionString("antani", "xx"); err == nil {
			t.Fatal("expected an error here")
		}
	})
}

type fakeExperimentConfig struct {
	Map    map[string]string
	String string
	Truth  bool
	Value  int64
}

func TestExperimentBuilderSetOptionAny(t *testing.T) {
	b := &ExperimentBuilder{config: &fakeExperimentConfig{}}

	t.Run("for missing field", func(t *testing.T) {
		if err := b.SetOptionAny("Antani", true); err == nil {
			t.Fatal("expected an error here")
		}
	})

	t.Run("for boolean", func(t *testing.T) {
		if err := b.SetOptionAny("Truth", "true"); err != nil {
			t.Fatal(err)
		}
		if err := b.SetOptionAny("Truth", "false"); err != nil {
			t.Fatal(err)
		}
		if err := b.SetOptionAny("Truth", false); err != nil {
			t.Fatal(err)
		}
		if err := b.SetOptionAny("Truth", "1234"); err == nil {
			t.Fatal("expected an error here")
		}
		if err := b.SetOptionAny("Truth", "yoloyolo"); err == nil {
			t.Fatal("expected an error here")
		}
		if err := b.SetOptionAny("Truth", map[string]string{}); err == nil {
			t.Fatal("expected an error here")
		}
	})

	t.Run("for integer", func(t *testing.T) {
		if err := b.SetOptionAny("Value", "true"); err == nil {
			t.Fatal("expected an error here")
		}
		if err := b.SetOptionAny("Value", "false"); err == nil {
			t.Fatal("expected an error here")
		}
		if err := b.SetOptionAny("Value", 1234); err != nil {
			t.Fatal(err)
		}
		if err := b.SetOptionAny("Value", int64(1234)); err != nil {
			t.Fatal(err)
		}
		if err := b.SetOptionAny("Value", int32(1234)); err != nil {
			t.Fatal(err)
		}
		if err := b.SetOptionAny("Value", int16(1234)); err != nil {
			t.Fatal(err)
		}
		if err := b.SetOptionAny("Value", int8(123)); err != nil {
			t.Fatal(err)
		}
		if err := b.SetOptionAny("Value", "1234"); err != nil {
			t.Fatal(err)
		}
		if err := b.SetOptionAny("Value", "yoloyolo"); err == nil {
			t.Fatal("expected an error here")
		}
		if err := b.SetOptionAny("Value", map[string]string{}); err == nil {
			t.Fatal("expected an error here")
		}
	})

	t.Run("for string", func(t *testing.T) {
		if err := b.SetOptionAny("String", "true"); err != nil {
			t.Fatal(err)
		}
		if err := b.SetOptionAny("String", "false"); err != nil {
			t.Fatal(err)
		}
		if err := b.SetOptionAny("String", "1234"); err != nil {
			t.Fatal(err)
		}
		if err := b.SetOptionAny("String", map[string]string{}); err == nil {
			t.Fatal("expected an error here")
		}
		if err := b.SetOptionAny("String", "yoloyolo"); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("for any other type", func(t *testing.T) {
		if err := b.SetOptionAny("Map", true); err == nil {
			t.Fatal("expected an error here")
		}
	})
}

func TestSetOptionsGuessType(t *testing.T) {
	b := &ExperimentBuilder{config: &fakeExperimentConfig{}}

	t.Run("we correctly handle an empty map", func(t *testing.T) {
		if err := b.SetOptionsAny(nil); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("we correctly handle a map containing options", func(t *testing.T) {
		f := &fakeExperimentConfig{}
		privateb := &ExperimentBuilder{config: f}
		opts := map[string]any{
			"String": "yoloyolo",
			"Value":  "174",
			"Truth":  "true",
		}
		if err := privateb.SetOptionsAny(opts); err != nil {
			t.Fatal(err)
		}
		if f.String != "yoloyolo" {
			t.Fatal("cannot set string value")
		}
		if f.Value != 174 {
			t.Fatal("cannot set integer value")
		}
		if f.Truth != true {
			t.Fatal("cannot set bool value")
		}
	})

	t.Run("we handle mistakes in a map containing options", func(t *testing.T) {
		opts := map[string]any{
			"String": "yoloyolo",
			"Value":  "antani;",
			"Truth":  "true",
		}
		if err := b.SetOptionsAny(opts); err == nil {
			t.Fatal("expected an error here")
		}
	})
}
