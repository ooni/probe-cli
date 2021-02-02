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

func TestExperimentBuilderSetOptionGuessType(t *testing.T) {
	type fiction struct {
		String string
		Truth  bool
		Value  int64
	}
	b := &ExperimentBuilder{config: &fiction{}}
	t.Run("we correctly guess a boolean", func(t *testing.T) {
		if err := b.SetOptionGuessType("Truth", "true"); err != nil {
			t.Fatal(err)
		}
		if err := b.SetOptionGuessType("Truth", "false"); err != nil {
			t.Fatal(err)
		}
		if err := b.SetOptionGuessType("Truth", "1234"); err == nil {
			t.Fatal("expected an error here")
		}
		if err := b.SetOptionGuessType("Truth", "yoloyolo"); err == nil {
			t.Fatal("expected an error here")
		}
	})
	t.Run("we correctly guess an integer", func(t *testing.T) {
		if err := b.SetOptionGuessType("Value", "true"); err == nil {
			t.Fatal("expected an error here")
		}
		if err := b.SetOptionGuessType("Value", "false"); err == nil {
			t.Fatal("expected an error here")
		}
		if err := b.SetOptionGuessType("Value", "1234"); err != nil {
			t.Fatal(err)
		}
		if err := b.SetOptionGuessType("Value", "yoloyolo"); err == nil {
			t.Fatal("expected an error here")
		}
	})
	t.Run("we correctly guess a string", func(t *testing.T) {
		if err := b.SetOptionGuessType("String", "true"); err == nil {
			t.Fatal("expected an error here")
		}
		if err := b.SetOptionGuessType("String", "false"); err == nil {
			t.Fatal("expected an error here")
		}
		if err := b.SetOptionGuessType("String", "1234"); err == nil {
			t.Fatal("expected an error here")
		}
		if err := b.SetOptionGuessType("String", "yoloyolo"); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("we correctly handle an empty map", func(t *testing.T) {
		if err := b.SetOptionsGuessType(nil); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("we correctly handle a map containing options", func(t *testing.T) {
		f := &fiction{}
		privateb := &ExperimentBuilder{config: f}
		opts := map[string]string{
			"String": "yoloyolo",
			"Value":  "174",
			"Truth":  "true",
		}
		if err := privateb.SetOptionsGuessType(opts); err != nil {
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
		opts := map[string]string{
			"String": "yoloyolo",
			"Value":  "antani;",
			"Truth":  "true",
		}
		if err := b.SetOptionsGuessType(opts); err == nil {
			t.Fatal("expected an error here")
		}
	})
}
