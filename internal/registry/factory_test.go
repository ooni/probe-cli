package registry

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
)

type fakeExperimentConfig struct {
	Chan   chan any `ooni:"we cannot set this"`
	String string   `ooni:"a string"`
	Truth  bool     `ooni:"something that no-one knows"`
	Value  int64    `ooni:"a number"`
}

func TestExperimentBuilderOptions(t *testing.T) {
	t.Run("when config is not a pointer", func(t *testing.T) {
		b := &Factory{
			config: 17,
		}
		options, err := b.Options()
		if !errors.Is(err, ErrConfigIsNotAStructPointer) {
			t.Fatal("expected an error here")
		}
		if options != nil {
			t.Fatal("expected nil here")
		}
	})

	t.Run("when config is not a struct", func(t *testing.T) {
		number := 17
		b := &Factory{
			config: &number,
		}
		options, err := b.Options()
		if !errors.Is(err, ErrConfigIsNotAStructPointer) {
			t.Fatal("expected an error here")
		}
		if options != nil {
			t.Fatal("expected nil here")
		}
	})

	t.Run("when config is a pointer to struct", func(t *testing.T) {
		config := &fakeExperimentConfig{}
		b := &Factory{
			config: config,
		}
		options, err := b.Options()
		if err != nil {
			t.Fatal(err)
		}
		for name, value := range options {
			switch name {
			case "Chan":
				if value.Doc != "we cannot set this" {
					t.Fatal("invalid doc")
				}
				if value.Type != "chan interface {}" {
					t.Fatal("invalid type", value.Type)
				}
			case "String":
				if value.Doc != "a string" {
					t.Fatal("invalid doc")
				}
				if value.Type != "string" {
					t.Fatal("invalid type", value.Type)
				}
			case "Truth":
				if value.Doc != "something that no-one knows" {
					t.Fatal("invalid doc")
				}
				if value.Type != "bool" {
					t.Fatal("invalid type", value.Type)
				}
			case "Value":
				if value.Doc != "a number" {
					t.Fatal("invalid doc")
				}
				if value.Type != "int64" {
					t.Fatal("invalid type", value.Type)
				}
			default:
				t.Fatal("unknown name", name)
			}
		}
	})
}

func TestExperimentBuilderSetOptionAny(t *testing.T) {
	var inputs = []struct {
		TestCaseName  string
		InitialConfig any
		FieldName     string
		FieldValue    any
		ExpectErr     error
		ExpectConfig  any
	}{{
		TestCaseName:  "config is not a pointer",
		InitialConfig: fakeExperimentConfig{},
		FieldName:     "Antani",
		FieldValue:    true,
		ExpectErr:     ErrConfigIsNotAStructPointer,
		ExpectConfig:  fakeExperimentConfig{},
	}, {
		TestCaseName: "config is not a pointer to struct",
		InitialConfig: func() *int {
			v := 17
			return &v
		}(),
		FieldName:  "Antani",
		FieldValue: true,
		ExpectErr:  ErrConfigIsNotAStructPointer,
		ExpectConfig: func() *int {
			v := 17
			return &v
		}(),
	}, {
		TestCaseName:  "for missing field",
		InitialConfig: &fakeExperimentConfig{},
		FieldName:     "Antani",
		FieldValue:    true,
		ExpectErr:     ErrNoSuchField,
		ExpectConfig:  &fakeExperimentConfig{},
	}, {
		TestCaseName:  "[bool] for true value represented as string",
		InitialConfig: &fakeExperimentConfig{},
		FieldName:     "Truth",
		FieldValue:    "true",
		ExpectErr:     nil,
		ExpectConfig: &fakeExperimentConfig{
			Truth: true,
		},
	}, {
		TestCaseName: "[bool] for false value represented as string",
		InitialConfig: &fakeExperimentConfig{
			Truth: true,
		},
		FieldName:  "Truth",
		FieldValue: "false",
		ExpectErr:  nil,
		ExpectConfig: &fakeExperimentConfig{
			Truth: false, // must have been flipped
		},
	}, {
		TestCaseName:  "[bool] for true value",
		InitialConfig: &fakeExperimentConfig{},
		FieldName:     "Truth",
		FieldValue:    true,
		ExpectErr:     nil,
		ExpectConfig: &fakeExperimentConfig{
			Truth: true,
		},
	}, {
		TestCaseName: "[bool] for false value",
		InitialConfig: &fakeExperimentConfig{
			Truth: true,
		},
		FieldName:  "Truth",
		FieldValue: false,
		ExpectErr:  nil,
		ExpectConfig: &fakeExperimentConfig{
			Truth: false, // must have been flipped
		},
	}, {
		TestCaseName:  "[bool] for invalid string representation of bool",
		InitialConfig: &fakeExperimentConfig{},
		FieldName:     "Truth",
		FieldValue:    "xxx",
		ExpectErr:     ErrInvalidStringRepresentationOfBool,
		ExpectConfig:  &fakeExperimentConfig{},
	}, {
		TestCaseName:  "[bool] for value we don't know how to convert to bool",
		InitialConfig: &fakeExperimentConfig{},
		FieldName:     "Truth",
		FieldValue:    make(chan any),
		ExpectErr:     ErrCannotSetBoolOption,
		ExpectConfig:  &fakeExperimentConfig{},
	}, {
		TestCaseName:  "[int] for int",
		InitialConfig: &fakeExperimentConfig{},
		FieldName:     "Value",
		FieldValue:    17,
		ExpectErr:     nil,
		ExpectConfig: &fakeExperimentConfig{
			Value: 17,
		},
	}, {
		TestCaseName:  "[int] for int64",
		InitialConfig: &fakeExperimentConfig{},
		FieldName:     "Value",
		FieldValue:    int64(17),
		ExpectErr:     nil,
		ExpectConfig: &fakeExperimentConfig{
			Value: 17,
		},
	}, {
		TestCaseName:  "[int] for int32",
		InitialConfig: &fakeExperimentConfig{},
		FieldName:     "Value",
		FieldValue:    int32(17),
		ExpectErr:     nil,
		ExpectConfig: &fakeExperimentConfig{
			Value: 17,
		},
	}, {
		TestCaseName:  "[int] for int16",
		InitialConfig: &fakeExperimentConfig{},
		FieldName:     "Value",
		FieldValue:    int16(17),
		ExpectErr:     nil,
		ExpectConfig: &fakeExperimentConfig{
			Value: 17,
		},
	}, {
		TestCaseName:  "[int] for int8",
		InitialConfig: &fakeExperimentConfig{},
		FieldName:     "Value",
		FieldValue:    int8(17),
		ExpectErr:     nil,
		ExpectConfig: &fakeExperimentConfig{
			Value: 17,
		},
	}, {
		TestCaseName:  "[int] for string representation of int",
		InitialConfig: &fakeExperimentConfig{},
		FieldName:     "Value",
		FieldValue:    "17",
		ExpectErr:     nil,
		ExpectConfig: &fakeExperimentConfig{
			Value: 17,
		},
	}, {
		TestCaseName:  "[int] for invalid string representation of int",
		InitialConfig: &fakeExperimentConfig{},
		FieldName:     "Value",
		FieldValue:    "xx",
		ExpectErr:     ErrCannotSetIntegerOption,
		ExpectConfig:  &fakeExperimentConfig{},
	}, {
		TestCaseName:  "[int] for type we don't know how to convert to int",
		InitialConfig: &fakeExperimentConfig{},
		FieldName:     "Value",
		FieldValue:    make(chan any),
		ExpectErr:     ErrCannotSetIntegerOption,
		ExpectConfig:  &fakeExperimentConfig{},
	}, {
		TestCaseName:  "[string] for serialized bool value while setting a string value",
		InitialConfig: &fakeExperimentConfig{},
		FieldName:     "String",
		FieldValue:    "true",
		ExpectErr:     nil,
		ExpectConfig: &fakeExperimentConfig{
			String: "true",
		},
	}, {
		TestCaseName:  "[string] for serialized int value while setting a string value",
		InitialConfig: &fakeExperimentConfig{},
		FieldName:     "String",
		FieldValue:    "155",
		ExpectErr:     nil,
		ExpectConfig: &fakeExperimentConfig{
			String: "155",
		},
	}, {
		TestCaseName:  "[string] for any other string",
		InitialConfig: &fakeExperimentConfig{},
		FieldName:     "String",
		FieldValue:    "xxx",
		ExpectErr:     nil,
		ExpectConfig: &fakeExperimentConfig{
			String: "xxx",
		},
	}, {
		TestCaseName:  "[string] for type we don't know how to convert to string",
		InitialConfig: &fakeExperimentConfig{},
		FieldName:     "String",
		FieldValue:    make(chan any),
		ExpectErr:     ErrCannotSetStringOption,
		ExpectConfig:  &fakeExperimentConfig{},
	}, {
		TestCaseName:  "for a field that we don't know how to set",
		InitialConfig: &fakeExperimentConfig{},
		FieldName:     "Chan",
		FieldValue:    make(chan any),
		ExpectErr:     ErrUnsupportedOptionType,
		ExpectConfig:  &fakeExperimentConfig{},
	}}

	for _, input := range inputs {
		t.Run(input.TestCaseName, func(t *testing.T) {
			ec := input.InitialConfig
			b := &Factory{config: ec}
			err := b.SetOptionAny(input.FieldName, input.FieldValue)
			if !errors.Is(err, input.ExpectErr) {
				t.Fatal(err)
			}
			if diff := cmp.Diff(input.ExpectConfig, ec); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestExperimentBuilderSetOptionsAny(t *testing.T) {
	b := &Factory{config: &fakeExperimentConfig{}}

	t.Run("we correctly handle an empty map", func(t *testing.T) {
		if err := b.SetOptionsAny(nil); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("we correctly handle a map containing options", func(t *testing.T) {
		f := &fakeExperimentConfig{}
		privateb := &Factory{config: f}
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

	t.Run("we handle mistakes in a map containing string options", func(t *testing.T) {
		opts := map[string]any{
			"String": "yoloyolo",
			"Value":  "xx",
			"Truth":  "true",
		}
		if err := b.SetOptionsAny(opts); !errors.Is(err, ErrCannotSetIntegerOption) {
			t.Fatal("unexpected err", err)
		}
	})
}
