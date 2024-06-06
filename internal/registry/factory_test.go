package registry

import (
	"errors"
	"fmt"
	"math"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/checkincache"
	"github.com/ooni/probe-cli/v3/internal/experiment/webconnectivitylte"
	"github.com/ooni/probe-cli/v3/internal/kvstore"
	"github.com/ooni/probe-cli/v3/internal/model"
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
		TestCaseName:  "[int] for NaN",
		InitialConfig: &fakeExperimentConfig{},
		FieldName:     "Value",
		FieldValue:    math.NaN(),
		ExpectErr:     ErrCannotSetIntegerOption,
		ExpectConfig:  &fakeExperimentConfig{},
	}, {
		TestCaseName:  "[int] for +Inf",
		InitialConfig: &fakeExperimentConfig{},
		FieldName:     "Value",
		FieldValue:    math.Inf(1),
		ExpectErr:     ErrCannotSetIntegerOption,
		ExpectConfig:  &fakeExperimentConfig{},
	}, {
		TestCaseName:  "[int] for -Inf",
		InitialConfig: &fakeExperimentConfig{},
		FieldName:     "Value",
		FieldValue:    math.Inf(-1),
		ExpectErr:     ErrCannotSetIntegerOption,
		ExpectConfig:  &fakeExperimentConfig{},
	}, {
		TestCaseName:  "[int] for too large value",
		InitialConfig: &fakeExperimentConfig{},
		FieldName:     "Value",
		FieldValue:    float64(jsonMaxInteger + 1),
		ExpectErr:     ErrCannotSetIntegerOption,
		ExpectConfig:  &fakeExperimentConfig{},
	}, {
		TestCaseName:  "[int] for too small value",
		InitialConfig: &fakeExperimentConfig{},
		FieldName:     "Value",
		FieldValue:    float64(jsonMinInteger - 1),
		ExpectErr:     ErrCannotSetIntegerOption,
		ExpectConfig:  &fakeExperimentConfig{},
	}, {
		TestCaseName:  "[int] for float64 with nonzero fractional value",
		InitialConfig: &fakeExperimentConfig{},
		FieldName:     "Value",
		FieldValue:    1.11,
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

func TestNewFactory(t *testing.T) {
	// experimentSpecificExpectations contains expectations for an experiment
	type experimentSpecificExpectations struct {
		// enabledByDefault contains the expected value for the enabledByDefault factory field.
		enabledByDefault bool

		// inputPolicy contains the expected value for the input policy.
		inputPolicy model.InputPolicy

		// interruptible contains the expected value for interrupted.
		interruptible bool
	}

	// expectationsMap contains expectations for each experiment that exists
	expectationsMap := map[string]*experimentSpecificExpectations{
		"dash": {
			enabledByDefault: true,
			inputPolicy:      model.InputNone,
			interruptible:    true,
		},
		"dnscheck": {
			enabledByDefault: true,
			inputPolicy:      model.InputOrStaticDefault,
		},
		"dnsping": {
			enabledByDefault: true,
			inputPolicy:      model.InputOrStaticDefault,
		},
		"echcheck": {
			// Note: echcheck is not enabled by default because we just introduced it
			// into 3.19.0-alpha, which makes it a relatively new experiment.
			//enabledByDefault: false,
			inputPolicy: model.InputOptional,
		},
		"example": {
			enabledByDefault: true,
			inputPolicy:      model.InputNone,
			interruptible:    true,
		},
		"facebook_messenger": {
			enabledByDefault: true,
			inputPolicy:      model.InputNone,
		},
		"http_header_field_manipulation": {
			enabledByDefault: true,
			inputPolicy:      model.InputNone,
		},
		"http_host_header": {
			enabledByDefault: true,
			inputPolicy:      model.InputOrQueryBackend,
		},
		"http_invalid_request_line": {
			enabledByDefault: true,
			inputPolicy:      model.InputNone,
		},
		"ndt": {
			enabledByDefault: true,
			inputPolicy:      model.InputNone,
			interruptible:    true,
		},
		"openvpn": {
			enabledByDefault: true,
			inputPolicy:      model.InputOrQueryBackend,
			interruptible:    true,
		},
		"portfiltering": {
			enabledByDefault: true,
			inputPolicy:      model.InputNone,
		},
		"psiphon": {
			enabledByDefault: true,
			inputPolicy:      model.InputOptional,
		},
		"quicping": {
			enabledByDefault: true,
			inputPolicy:      model.InputStrictlyRequired,
		},
		"riseupvpn": {
			// Note: riseupvpn is not enabled by default because it has been flaky
			// in the past and we want to be defensive here.
			//enabledByDefault: false,
			inputPolicy: model.InputNone,
		},
		"run": {
			enabledByDefault: true,
			inputPolicy:      model.InputStrictlyRequired,
		},
		"signal": {
			enabledByDefault: true,
			inputPolicy:      model.InputNone,
		},
		"simple_sni": {
			// Note: simple_sni is not enabled by default because it has only been
			// introduced for writing tutorials and should not be used.
			//enabledByDefault: false,
			inputPolicy: model.InputOrQueryBackend,
		},
		"simplequicping": {
			enabledByDefault: true,
			inputPolicy:      model.InputStrictlyRequired,
		},
		"sni_blocking": {
			enabledByDefault: true,
			inputPolicy:      model.InputOrQueryBackend,
		},
		"stunreachability": {
			enabledByDefault: true,
			inputPolicy:      model.InputOrStaticDefault,
		},
		"tcpping": {
			enabledByDefault: true,
			inputPolicy:      model.InputStrictlyRequired,
		},
		"tlsmiddlebox": {
			enabledByDefault: true,
			inputPolicy:      model.InputStrictlyRequired,
		},
		"telegram": {
			enabledByDefault: true,
			inputPolicy:      model.InputNone,
		},
		"tlsping": {
			enabledByDefault: true,
			inputPolicy:      model.InputStrictlyRequired,
		},
		"tlstool": {
			enabledByDefault: true,
			inputPolicy:      model.InputOrQueryBackend,
		},
		"tor": {
			enabledByDefault: true,
			inputPolicy:      model.InputNone,
		},
		"torsf": {
			// We suspect there will be changes in torsf SNI soon. We are not prepared to
			// serve these changes using the check-in API. Hence, disable torsf by default
			// and require enabling it using the check-in API feature flags.
			//enabledByDefault: false,
			inputPolicy: model.InputNone,
		},
		"urlgetter": {
			enabledByDefault: true,
			inputPolicy:      model.InputStrictlyRequired,
		},
		"vanilla_tor": {
			// The experiment crashes on Android and possibly also iOS. We want to
			// control whether and when to run it using check-in.
			//enabledByDefault: false,
			inputPolicy: model.InputNone,
		},
		"web_connectivity": {
			enabledByDefault: true,
			inputPolicy:      model.InputOrQueryBackend,
		},
		"web_connectivity@v0.5": {
			enabledByDefault: true,
			inputPolicy:      model.InputOrQueryBackend,
		},
		"whatsapp": {
			enabledByDefault: true,
			inputPolicy:      model.InputNone,
		},
	}

	// testCase is a test case checked by this func
	type testCase struct {
		// description describes the test case
		description string

		// experimentName is the experiment experimentName
		experimentName string

		// kvStore is the key-value store to use
		kvStore model.KeyValueStore

		// setForceEnableExperiment sets the OONI_FORCE_ENABLE_EXPERIMENT=1 env variable
		setForceEnableExperiment bool

		// expectErr is the error we expect when calling NewFactory
		expectErr error
	}

	// allCases contains all test cases
	allCases := []*testCase{}

	// create test cases for canonical experiment names
	for _, name := range ExperimentNames() {
		allCases = append(allCases, &testCase{
			description:    name,
			experimentName: name,
			kvStore:        &kvstore.Memory{},
			expectErr: (func() error {
				expectations := expectationsMap[name]
				if expectations == nil {
					t.Fatal("no expectations for", name)
				}
				if !expectations.enabledByDefault {
					return ErrRequiresForceEnable
				}
				return nil
			}()),
		})
	}

	// add additional test for the ndt7 experiment name
	allCases = append(allCases, &testCase{
		description:    "the ndt7 name still works",
		experimentName: "ndt7",
		kvStore:        &kvstore.Memory{},
		expectErr:      nil,
	})

	// add additional test for the dns_check experiment name
	allCases = append(allCases, &testCase{
		description:    "the dns_check name still works",
		experimentName: "dns_check",
		kvStore:        &kvstore.Memory{},
		expectErr:      nil,
	})

	// add additional test for the stun_reachability experiment name
	allCases = append(allCases, &testCase{
		description:    "the stun_reachability name still works",
		experimentName: "stun_reachability",
		kvStore:        &kvstore.Memory{},
		expectErr:      nil,
	})

	// add additional test for the web_connectivity@v_0_5 experiment name
	allCases = append(allCases, &testCase{
		description:    "the web_connectivity@v_0_5 name still works",
		experimentName: "web_connectivity@v_0_5",
		kvStore:        &kvstore.Memory{},
		expectErr:      nil,
	})

	// make sure we can create default-not-enabled experiments if we
	// configure the proper environment variable
	for name, expectations := range expectationsMap {
		if expectations.enabledByDefault {
			continue
		}

		allCases = append(allCases, &testCase{
			description:              fmt.Sprintf("we can create %s with OONI_FORCE_ENABLE_EXPERIMENT=1", name),
			experimentName:           name,
			kvStore:                  &kvstore.Memory{},
			setForceEnableExperiment: true,
			expectErr:                nil,
		})
	}

	// make sure we can create default-not-enabled experiments if we
	// configure the proper check-in flags
	for name, expectations := range expectationsMap {
		if expectations.enabledByDefault {
			continue
		}

		// create a check-in configuration with the experiment being enabled
		store := &kvstore.Memory{}
		checkincache.Store(store, &model.OOAPICheckInResult{
			Conf: model.OOAPICheckInResultConfig{
				Features: map[string]bool{
					checkincache.ExperimentEnabledKey(name): true,
				},
			},
		})

		allCases = append(allCases, &testCase{
			description:              fmt.Sprintf("we can create %s with the proper check-in config", name),
			experimentName:           name,
			kvStore:                  store,
			setForceEnableExperiment: false,
			expectErr:                nil,
		})
	}

	// perform checks for each name
	for _, tc := range allCases {
		t.Run(tc.description, func(t *testing.T) {
			// make sure the bypass environment variable is not set
			if os.Getenv(OONI_FORCE_ENABLE_EXPERIMENT) != "" {
				t.Fatal("the OONI_FORCE_ENABLE_EXPERIMENT env variable shouldn't be set")
			}

			// if needed, set the environment variable for the scope of the func
			if tc.setForceEnableExperiment {
				os.Setenv(OONI_FORCE_ENABLE_EXPERIMENT, "1")
				defer os.Unsetenv(OONI_FORCE_ENABLE_EXPERIMENT)
			}

			t.Log("experimentName:", tc.experimentName)

			// get experiment expectations -- note that here we must canonicalize the
			// experiment name otherwise we won't find it into the map when testing non-canonical names
			expectations := expectationsMap[CanonicalizeExperimentName(tc.experimentName)]
			if expectations == nil {
				t.Fatal("no expectations for", tc.experimentName)
			}

			t.Logf("expectations: %+v", expectations)

			// get the experiment factory
			factory, err := NewFactory(tc.experimentName, tc.kvStore, model.DiscardLogger)

			t.Logf("NewFactory returned: %+v %+v", factory, err)

			// make sure the returned error makes sense
			switch {
			case tc.expectErr == nil && err != nil:
				t.Fatal(tc.experimentName, ": expected", tc.expectErr, "got", err)

			case tc.expectErr != nil && err == nil:
				t.Fatal(tc.experimentName, ": expected", tc.expectErr, "got", err)

			case tc.expectErr != nil && err != nil:
				if !errors.Is(err, tc.expectErr) {
					t.Fatal(tc.experimentName, ": expected", tc.expectErr, "got", err)
				}
				return

			case tc.expectErr == nil && err == nil:
				// fallthrough
			}

			// make sure the enabled by default field is consistent with expectations
			if factory.enabledByDefault != expectations.enabledByDefault {
				t.Fatal(tc.experimentName, ": expected", expectations.enabledByDefault, "got", factory.enabledByDefault)
			}

			// make sure the input policy is the expected one
			if v := factory.InputPolicy(); v != expectations.inputPolicy {
				t.Fatal(tc.experimentName, ": expected", expectations.inputPolicy, "got", v)
			}

			// make sure the interruptible value is the expected one
			if v := factory.Interruptible(); v != expectations.interruptible {
				t.Fatal(tc.experimentName, ": expected", expectations.interruptible, "got", v)
			}

			// make sure we can create the measurer
			measurer := factory.NewExperimentMeasurer()
			if measurer == nil {
				t.Fatal("expected non-nil measurer, got nil")
			}
		})
	}

	// make sure we create web_connectivity@v0.5 when the check-in says so
	t.Run("we honor check-in flags for web_connectivity@v0.5", func(t *testing.T) {
		// create a keyvalue store with the proper flags
		store := &kvstore.Memory{}
		checkincache.Store(store, &model.OOAPICheckInResult{
			Conf: model.OOAPICheckInResultConfig{
				Features: map[string]bool{
					"webconnectivity_0.5": true,
				},
			},
		})

		// get the experiment factory
		factory, err := NewFactory("web_connectivity", store, model.DiscardLogger)
		if err != nil {
			t.Fatal(err)
		}

		// make sure the enabled by default field is consistent with expectations
		if !factory.enabledByDefault {
			t.Fatal("expected enabledByDefault to be true")
		}

		// make sure the input policy is the expected one
		if factory.InputPolicy() != model.InputOrQueryBackend {
			t.Fatal("expected inputPolicy to be InputOrQueryBackend")
		}

		// make sure the interrupted value is the expected one
		if factory.Interruptible() {
			t.Fatal("expected interruptible to be false")
		}

		// make sure we can create the measurer
		measurer := factory.NewExperimentMeasurer()
		if measurer == nil {
			t.Fatal("expected non-nil measurer, got nil")
		}

		// make sure the type we're creating is the correct one
		if _, good := measurer.(*webconnectivitylte.Measurer); !good {
			t.Fatalf("expected to see an instance of *webconnectivitylte.Measurer, got %T", measurer)
		}
	})

	// add a test case for a nonexistent experiment
	t.Run("we correctly return an error for a nonexistent experiment", func(t *testing.T) {
		// the empty string is a nonexistent experiment
		factory, err := NewFactory("", &kvstore.Memory{}, model.DiscardLogger)
		if !errors.Is(err, ErrNoSuchExperiment) {
			t.Fatal("unexpected err", err)
		}
		if factory != nil {
			t.Fatal("expected nil factory here")
		}
	})
}
