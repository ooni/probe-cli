package engine

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/apex/log"
	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/kvstore"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

func TestInputLoaderInputNoneWithStaticInputs(t *testing.T) {
	il := &InputLoader{
		StaticInputs: []string{"https://www.google.com/"},
		InputPolicy:  model.InputNone,
	}
	ctx := context.Background()
	out, err := il.Load(ctx)
	if !errors.Is(err, ErrNoInputExpected) {
		t.Fatalf("not the error we expected: %+v", err)
	}
	if out != nil {
		t.Fatal("not the output we expected")
	}
}

func TestInputLoaderInputNoneWithFilesInputs(t *testing.T) {
	il := &InputLoader{
		SourceFiles: []string{
			"testdata/inputloader1.txt",
			"testdata/inputloader2.txt",
		},
		InputPolicy: model.InputNone,
	}
	ctx := context.Background()
	out, err := il.Load(ctx)
	if !errors.Is(err, ErrNoInputExpected) {
		t.Fatalf("not the error we expected: %+v", err)
	}
	if out != nil {
		t.Fatal("not the output we expected")
	}
}

func TestInputLoaderInputNoneWithBothInputs(t *testing.T) {
	il := &InputLoader{
		StaticInputs: []string{"https://www.google.com/"},
		SourceFiles: []string{
			"testdata/inputloader1.txt",
			"testdata/inputloader2.txt",
		},
		InputPolicy: model.InputNone,
	}
	ctx := context.Background()
	out, err := il.Load(ctx)
	if !errors.Is(err, ErrNoInputExpected) {
		t.Fatalf("not the error we expected: %+v", err)
	}
	if out != nil {
		t.Fatal("not the output we expected")
	}
}

func TestInputLoaderInputNoneWithNoInput(t *testing.T) {
	il := &InputLoader{
		InputPolicy: model.InputNone,
	}
	ctx := context.Background()
	out, err := il.Load(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0].Input() != "" {
		t.Fatal("not the output we expected")
	}
}

func TestInputLoaderInputOptionalWithNoInput(t *testing.T) {
	il := &InputLoader{
		InputPolicy: model.InputOptional,
	}
	ctx := context.Background()
	out, err := il.Load(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0].Input() != "" {
		t.Fatal("not the output we expected")
	}
}

func TestInputLoaderInputOptionalWithInput(t *testing.T) {
	il := &InputLoader{
		StaticInputs: []string{"https://www.google.com/"},
		SourceFiles: []string{
			"testdata/inputloader1.txt",
			"testdata/inputloader2.txt",
		},
		InputPolicy: model.InputOptional,
	}
	ctx := context.Background()
	out, err := il.Load(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 5 {
		t.Fatal("not the output length we expected")
	}
	expect := []model.ExperimentTarget{
		&model.OOAPIURLInfo{
			CountryCode:  model.DefaultCountryCode,
			CategoryCode: model.DefaultCategoryCode,
			URL:          "https://www.google.com/",
		},
		&model.OOAPIURLInfo{
			CountryCode:  model.DefaultCountryCode,
			CategoryCode: model.DefaultCategoryCode,
			URL:          "https://www.x.org/",
		},
		&model.OOAPIURLInfo{
			CountryCode:  model.DefaultCountryCode,
			CategoryCode: model.DefaultCategoryCode,
			URL:          "https://www.slashdot.org/",
		},
		&model.OOAPIURLInfo{
			CountryCode:  model.DefaultCountryCode,
			CategoryCode: model.DefaultCategoryCode,
			URL:          "https://abc.xyz/",
		},
		&model.OOAPIURLInfo{
			CountryCode:  model.DefaultCountryCode,
			CategoryCode: model.DefaultCategoryCode,
			URL:          "https://run.ooni.io/",
		},
	}
	if diff := cmp.Diff(out, expect); diff != "" {
		t.Fatal(diff)
	}
}

func TestInputLoaderInputOptionalNonexistentFile(t *testing.T) {
	il := &InputLoader{
		StaticInputs: []string{"https://www.google.com/"},
		SourceFiles: []string{
			"testdata/inputloader1.txt",
			"/nonexistent",
			"testdata/inputloader2.txt",
		},
		InputPolicy: model.InputOptional,
	}
	ctx := context.Background()
	out, err := il.Load(ctx)
	if !errors.Is(err, syscall.ENOENT) {
		t.Fatalf("not the error we expected: %+v", err)
	}
	if out != nil {
		t.Fatal("not the output we expected")
	}
}

func TestInputLoaderInputStrictlyRequiredWithInput(t *testing.T) {
	il := &InputLoader{
		StaticInputs: []string{"https://www.google.com/"},
		SourceFiles: []string{
			"testdata/inputloader1.txt",
			"testdata/inputloader2.txt",
		},
		InputPolicy: model.InputStrictlyRequired,
	}
	ctx := context.Background()
	out, err := il.Load(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 5 {
		t.Fatal("not the output length we expected")
	}
	expect := []model.ExperimentTarget{
		&model.OOAPIURLInfo{
			CountryCode:  model.DefaultCountryCode,
			CategoryCode: model.DefaultCategoryCode,
			URL:          "https://www.google.com/",
		},
		&model.OOAPIURLInfo{
			CountryCode:  model.DefaultCountryCode,
			CategoryCode: model.DefaultCategoryCode,
			URL:          "https://www.x.org/",
		},
		&model.OOAPIURLInfo{
			CountryCode:  model.DefaultCountryCode,
			CategoryCode: model.DefaultCategoryCode,
			URL:          "https://www.slashdot.org/",
		},
		&model.OOAPIURLInfo{
			CountryCode:  model.DefaultCountryCode,
			CategoryCode: model.DefaultCategoryCode,
			URL:          "https://abc.xyz/",
		},
		&model.OOAPIURLInfo{
			CountryCode:  model.DefaultCountryCode,
			CategoryCode: model.DefaultCategoryCode,
			URL:          "https://run.ooni.io/",
		},
	}
	if diff := cmp.Diff(out, expect); diff != "" {
		t.Fatal(diff)
	}
}

func TestInputLoaderInputStrictlyRequiredWithoutInput(t *testing.T) {
	il := &InputLoader{
		InputPolicy: model.InputStrictlyRequired,
	}
	ctx := context.Background()
	out, err := il.Load(ctx)
	if !errors.Is(err, ErrInputRequired) {
		t.Fatalf("not the error we expected: %+v", err)
	}
	if out != nil {
		t.Fatal("not the output we expected")
	}
}

func TestInputLoaderInputStrictlyRequiredWithEmptyFile(t *testing.T) {
	il := &InputLoader{
		InputPolicy: model.InputStrictlyRequired,
		SourceFiles: []string{
			"testdata/inputloader1.txt",
			"testdata/inputloader3.txt", // we want it before inputloader2.txt
			"testdata/inputloader2.txt",
		},
	}
	ctx := context.Background()
	out, err := il.Load(ctx)
	if !errors.Is(err, ErrDetectedEmptyFile) {
		t.Fatalf("not the error we expected: %+v", err)
	}
	if out != nil {
		t.Fatal("not the output we expected")
	}
}

func TestInputLoaderInputOrStaticDefaultWithInput(t *testing.T) {
	il := &InputLoader{
		ExperimentName: "dnscheck",
		StaticInputs:   []string{"https://www.google.com/"},
		SourceFiles: []string{
			"testdata/inputloader1.txt",
			"testdata/inputloader2.txt",
		},
		InputPolicy: model.InputOrStaticDefault,
	}
	ctx := context.Background()
	out, err := il.Load(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 5 {
		t.Fatal("not the output length we expected")
	}
	expect := []model.ExperimentTarget{
		&model.OOAPIURLInfo{
			CountryCode:  model.DefaultCountryCode,
			CategoryCode: model.DefaultCategoryCode,
			URL:          "https://www.google.com/",
		},
		&model.OOAPIURLInfo{
			CountryCode:  model.DefaultCountryCode,
			CategoryCode: model.DefaultCategoryCode,
			URL:          "https://www.x.org/",
		},
		&model.OOAPIURLInfo{
			CountryCode:  model.DefaultCountryCode,
			CategoryCode: model.DefaultCategoryCode,
			URL:          "https://www.slashdot.org/",
		},
		&model.OOAPIURLInfo{
			CountryCode:  model.DefaultCountryCode,
			CategoryCode: model.DefaultCategoryCode,
			URL:          "https://abc.xyz/",
		},
		&model.OOAPIURLInfo{
			CountryCode:  model.DefaultCountryCode,
			CategoryCode: model.DefaultCategoryCode,
			URL:          "https://run.ooni.io/",
		},
	}
	if diff := cmp.Diff(out, expect); diff != "" {
		t.Fatal(diff)
	}
}

func TestInputLoaderInputOrStaticDefaultWithEmptyFile(t *testing.T) {
	il := &InputLoader{
		ExperimentName: "dnscheck",
		InputPolicy:    model.InputOrStaticDefault,
		SourceFiles: []string{
			"testdata/inputloader1.txt",
			"testdata/inputloader3.txt", // we want it before inputloader2.txt
			"testdata/inputloader2.txt",
		},
	}
	ctx := context.Background()
	out, err := il.Load(ctx)
	if !errors.Is(err, ErrDetectedEmptyFile) {
		t.Fatalf("not the error we expected: %+v", err)
	}
	if out != nil {
		t.Fatal("not the output we expected")
	}
}

func TestInputLoaderInputOrStaticDefaultWithoutInputDNSCheck(t *testing.T) {
	il := &InputLoader{
		ExperimentName: "dnscheck",
		InputPolicy:    model.InputOrStaticDefault,
	}
	ctx := context.Background()
	out, err := il.Load(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != len(dnsCheckDefaultInput) {
		t.Fatal("invalid output length")
	}
	for idx := 0; idx < len(dnsCheckDefaultInput); idx++ {
		e := out[idx]
		if e.Category() != model.DefaultCategoryCode {
			t.Fatal("invalid category code")
		}
		if e.Country() != model.DefaultCountryCode {
			t.Fatal("invalid country code")
		}
		if e.Input() != dnsCheckDefaultInput[idx] {
			t.Fatal("invalid URL")
		}
	}
}

func TestInputLoaderInputOrStaticDefaultWithoutInputStunReachability(t *testing.T) {
	il := &InputLoader{
		ExperimentName: "stunreachability",
		InputPolicy:    model.InputOrStaticDefault,
	}
	ctx := context.Background()
	out, err := il.Load(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != len(stunReachabilityDefaultInput) {
		t.Fatal("invalid output length")
	}
	for idx := 0; idx < len(stunReachabilityDefaultInput); idx++ {
		e := out[idx]
		if e.Category() != model.DefaultCategoryCode {
			t.Fatal("invalid category code")
		}
		if e.Country() != model.DefaultCountryCode {
			t.Fatal("invalid country code")
		}
		if e.Input() != stunReachabilityDefaultInput[idx] {
			t.Fatal("invalid URL")
		}
	}
}

func TestStaticBareInputForExperimentWorksWithNonCanonicalNames(t *testing.T) {
	names := []string{"DNSCheck", "STUNReachability"}
	for _, name := range names {
		if _, err := staticInputForExperiment(name); err != nil {
			t.Fatal("failure for", name, ":", err)
		}
	}
}

func TestInputLoaderInputOrStaticDefaultWithoutInputOtherName(t *testing.T) {
	il := &InputLoader{
		ExperimentName: "xx",
		InputPolicy:    model.InputOrStaticDefault,
	}
	ctx := context.Background()
	out, err := il.Load(ctx)
	if !errors.Is(err, ErrNoStaticInput) {
		t.Fatal("not the error we expected", err)
	}
	if out != nil {
		t.Fatal("expected nil result here")
	}
}

func TestInputLoaderInputOrQueryBackendWithInput(t *testing.T) {
	il := &InputLoader{
		StaticInputs: []string{"https://www.google.com/"},
		SourceFiles: []string{
			"testdata/inputloader1.txt",
			"testdata/inputloader2.txt",
		},
		InputPolicy: model.InputOrQueryBackend,
	}
	ctx := context.Background()
	out, err := il.Load(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 5 {
		t.Fatal("not the output length we expected")
	}
	expect := []model.ExperimentTarget{
		&model.OOAPIURLInfo{
			CountryCode:  model.DefaultCountryCode,
			CategoryCode: model.DefaultCategoryCode,
			URL:          "https://www.google.com/",
		},
		&model.OOAPIURLInfo{
			CountryCode:  model.DefaultCountryCode,
			CategoryCode: model.DefaultCategoryCode,
			URL:          "https://www.x.org/",
		},
		&model.OOAPIURLInfo{
			CountryCode:  model.DefaultCountryCode,
			CategoryCode: model.DefaultCategoryCode,
			URL:          "https://www.slashdot.org/",
		},
		&model.OOAPIURLInfo{
			CountryCode:  model.DefaultCountryCode,
			CategoryCode: model.DefaultCategoryCode,
			URL:          "https://abc.xyz/",
		},
		&model.OOAPIURLInfo{
			CountryCode:  model.DefaultCountryCode,
			CategoryCode: model.DefaultCategoryCode,
			URL:          "https://run.ooni.io/",
		},
	}
	if diff := cmp.Diff(out, expect); diff != "" {
		t.Fatal(diff)
	}
}

func TestInputLoaderInputOrQueryBackendWithNoInputAndCancelledContext(t *testing.T) {
	sess, err := NewSession(context.Background(), SessionConfig{
		KVStore:         &kvstore.Memory{},
		Logger:          log.Log,
		SoftwareName:    "miniooni",
		SoftwareVersion: "0.1.0-dev",
		TempDir:         "testdata",
	})
	if err != nil {
		t.Fatal(err)
	}
	defer sess.Close()
	il := &InputLoader{
		InputPolicy: model.InputOrQueryBackend,
		Session:     sess,
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // fail immediately
	out, err := il.Load(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("not the error we expected: %+v", err)
	}
	if out != nil {
		t.Fatal("not the output we expected")
	}
}

func TestInputLoaderInputOrQueryBackendWithEmptyFile(t *testing.T) {
	il := &InputLoader{
		InputPolicy: model.InputOrQueryBackend,
		SourceFiles: []string{
			"testdata/inputloader1.txt",
			"testdata/inputloader3.txt", // we want it before inputloader2.txt
			"testdata/inputloader2.txt",
		},
	}
	ctx := context.Background()
	out, err := il.Load(ctx)
	if !errors.Is(err, ErrDetectedEmptyFile) {
		t.Fatalf("not the error we expected: %+v", err)
	}
	if out != nil {
		t.Fatal("not the output we expected")
	}
}

type InputLoaderBrokenFS struct{}

func (InputLoaderBrokenFS) Open(filepath string) (fs.File, error) {
	return InputLoaderBrokenFile{}, nil
}

type InputLoaderBrokenFile struct{}

func (InputLoaderBrokenFile) Stat() (os.FileInfo, error) {
	return nil, nil
}

func (InputLoaderBrokenFile) Read([]byte) (int, error) {
	return 0, syscall.EFAULT
}

func (InputLoaderBrokenFile) Close() error {
	return nil
}

func TestInputLoaderReadfileScannerFailure(t *testing.T) {
	il := &InputLoader{}
	out, err := il.readfile("", InputLoaderBrokenFS{}.Open)
	if !errors.Is(err, syscall.EFAULT) {
		t.Fatal("not the error we expected")
	}
	if out != nil {
		t.Fatal("not the output we expected")
	}
}

// InputLoaderMockableSession is a mockable session
// used by InputLoader tests.
type InputLoaderMockableSession struct {
	// Output contains the output of CheckIn. It should
	// be nil when Error is not-nil.
	Output *model.OOAPICheckInResult

	// FetchOpenVPNConfigOutput contains the output of FetchOpenVPNConfig.
	// It should be nil when Error is non-nil.
	FetchOpenVPNConfigOutput *model.OOAPIVPNProviderConfig

	// Error is the error to be returned by CheckIn. It
	// should be nil when Output is not-nil.
	Error error
}

// CheckIn implements InputLoaderSession.CheckIn.
func (sess *InputLoaderMockableSession) CheckIn(
	ctx context.Context, config *model.OOAPICheckInConfig) (*model.OOAPICheckInResult, error) {
	if sess.Output == nil && sess.Error == nil {
		return nil, errors.New("both Output and Error are nil")
	}
	return sess.Output, sess.Error
}

// FetchOpenVPNConfig implements InputLoaderSession.FetchOpenVPNConfig.
func (sess *InputLoaderMockableSession) FetchOpenVPNConfig(
	ctx context.Context, provider, cc string) (*model.OOAPIVPNProviderConfig, error) {
	runtimex.Assert(!(sess.Error == nil && sess.FetchOpenVPNConfigOutput == nil), "both FetchOpenVPNConfig and Error are nil")
	return sess.FetchOpenVPNConfigOutput, sess.Error
}

func TestInputLoaderCheckInFailure(t *testing.T) {
	il := &InputLoader{
		Session: &InputLoaderMockableSession{
			Error: io.EOF,
		},
	}
	out, err := il.loadRemote(context.Background())
	if !errors.Is(err, io.EOF) {
		t.Fatalf("not the error we expected: %+v", err)
	}
	if out != nil {
		t.Fatal("expected nil output here")
	}
}

func TestInputLoaderCheckInSuccessWithNilWebConnectivity(t *testing.T) {
	il := &InputLoader{
		Session: &InputLoaderMockableSession{
			Output: &model.OOAPICheckInResult{
				Tests: model.OOAPICheckInResultNettests{},
			},
		},
	}
	out, err := il.loadRemote(context.Background())
	if !errors.Is(err, ErrNoURLsReturned) {
		t.Fatalf("not the error we expected: %+v", err)
	}
	if out != nil {
		t.Fatal("expected nil output here")
	}
}

func TestInputLoaderCheckInSuccessWithNoURLs(t *testing.T) {
	il := &InputLoader{
		Session: &InputLoaderMockableSession{
			Output: &model.OOAPICheckInResult{
				Tests: model.OOAPICheckInResultNettests{
					WebConnectivity: &model.OOAPICheckInInfoWebConnectivity{},
				},
			},
		},
	}
	out, err := il.loadRemote(context.Background())
	if !errors.Is(err, ErrNoURLsReturned) {
		t.Fatalf("not the error we expected: %+v", err)
	}
	if out != nil {
		t.Fatal("expected nil output here")
	}
}

func TestInputLoaderCheckInSuccessWithSomeURLs(t *testing.T) {
	inputs0 := model.OOAPIURLInfo{
		CategoryCode: "NEWS",
		CountryCode:  "IT",
		URL:          "https://repubblica.it",
	}
	inputs1 := model.OOAPIURLInfo{
		CategoryCode: "NEWS",
		CountryCode:  "IT",
		URL:          "https://corriere.it",
	}
	inputs := []model.OOAPIURLInfo{inputs0, inputs1}
	expect := []model.ExperimentTarget{&inputs0, &inputs1}
	il := &InputLoader{
		Session: &InputLoaderMockableSession{
			Output: &model.OOAPICheckInResult{
				Tests: model.OOAPICheckInResultNettests{
					WebConnectivity: &model.OOAPICheckInInfoWebConnectivity{
						URLs: inputs,
					},
				},
			},
		},
	}
	out, err := il.loadRemote(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(expect, out); diff != "" {
		t.Fatal(diff)
	}
}

func TestInputLoaderOpenVPNSuccessWithNoInput(t *testing.T) {
	il := &InputLoader{
		ExperimentName: "openvpn",
		InputPolicy:    model.InputOrQueryBackend,
		Session: &InputLoaderMockableSession{
			Error: nil,
			FetchOpenVPNConfigOutput: &model.OOAPIVPNProviderConfig{
				Provider: "riseup",
				Inputs: []string{
					"openvpn://foo.corp/?address=1.1.1.1:1194&transport=tcp",
				},
				DateUpdated: time.Now(),
			},
		},
	}
	_, err := il.loadRemote(context.Background())
	if err != nil {
		t.Fatal("we did not expect an error")
	}
}

func TestInputLoaderOpenVPNSuccessWithNoInputAndAPICall(t *testing.T) {
	il := &InputLoader{
		ExperimentName: "openvpn",
		InputPolicy:    model.InputOrQueryBackend,
		Session: &InputLoaderMockableSession{
			Error: nil,
			FetchOpenVPNConfigOutput: &model.OOAPIVPNProviderConfig{
				Provider: "riseupvpn",
				Inputs: []string{
					"openvpn://foo.corp/?address=1.2.3.4:1194&transport=tcp",
				},
				DateUpdated: time.Now(),
			},
		},
	}
	out, err := il.loadRemote(context.Background())
	if err != nil {
		t.Fatal("we did not expect an error")
	}
	if len(out) != 1 {
		t.Fatal("we expected output of len=1")
	}
}

func TestInputLoaderOpenVPNWithAPIFailureAndFallback(t *testing.T) {
	expected := errors.New("mocked API error")
	il := &InputLoader{
		ExperimentName: "openvpn",
		InputPolicy:    model.InputOrQueryBackend,
		Session: &InputLoaderMockableSession{
			Error: expected,
		},
	}
	out, err := il.loadRemote(context.Background())
	if err != expected {
		t.Fatal("we expected an error")
	}
	if len(out) != 0 {
		t.Fatal("we expected no fallback URLs")
	}
}

func TestPreventMistakesWithCategories(t *testing.T) {
	input := []model.OOAPIURLInfo{{
		CategoryCode: "NEWS",
		URL:          "https://repubblica.it/",
		CountryCode:  "IT",
	}, {
		CategoryCode: "HACK",
		URL:          "https://2600.com",
		CountryCode:  "XX",
	}, {
		CategoryCode: "FILE",
		URL:          "https://addons.mozilla.org/",
		CountryCode:  "XX",
	}}
	desired := []model.OOAPIURLInfo{{
		CategoryCode: "NEWS",
		URL:          "https://repubblica.it/",
		CountryCode:  "IT",
	}, {
		CategoryCode: "FILE",
		URL:          "https://addons.mozilla.org/",
		CountryCode:  "XX",
	}}
	il := &InputLoader{}
	output := il.preventMistakes(input, []string{"NEWS", "FILE"})
	if diff := cmp.Diff(desired, output); diff != "" {
		t.Fatal(diff)
	}
}

func TestPreventMistakesWithoutCategoriesAndNil(t *testing.T) {
	input := []model.OOAPIURLInfo{{
		CategoryCode: "NEWS",
		URL:          "https://repubblica.it/",
		CountryCode:  "IT",
	}, {
		CategoryCode: "HACK",
		URL:          "https://2600.com",
		CountryCode:  "XX",
	}, {
		CategoryCode: "FILE",
		URL:          "https://addons.mozilla.org/",
		CountryCode:  "XX",
	}}
	il := &InputLoader{}
	output := il.preventMistakes(input, nil)
	if diff := cmp.Diff(input, output); diff != "" {
		t.Fatal(diff)
	}
}

func TestPreventMistakesWithoutCategoriesAndEmpty(t *testing.T) {
	input := []model.OOAPIURLInfo{{
		CategoryCode: "NEWS",
		URL:          "https://repubblica.it/",
		CountryCode:  "IT",
	}, {
		CategoryCode: "HACK",
		URL:          "https://2600.com",
		CountryCode:  "XX",
	}, {
		CategoryCode: "FILE",
		URL:          "https://addons.mozilla.org/",
		CountryCode:  "XX",
	}}
	il := &InputLoader{}
	output := il.preventMistakes(input, []string{})
	if diff := cmp.Diff(input, output); diff != "" {
		t.Fatal(diff)
	}
}

// InputLoaderFakeLogger is a fake InputLoaderLogger.
type InputLoaderFakeLogger struct{}

// Warnf implements InputLoaderLogger.Warnf
func (ilfl *InputLoaderFakeLogger) Warnf(format string, v ...interface{}) {}

func TestInputLoaderLoggerWorksAsIntended(t *testing.T) {
	logger := &InputLoaderFakeLogger{}
	inputLoader := &InputLoader{Logger: logger}
	out := inputLoader.logger()
	if out != logger {
		t.Fatal("logger not working as intended")
	}
}

func TestStringListToModelURLInfoWithValidInput(t *testing.T) {
	input := []string{
		"stun://stun.voip.blackberry.com:3478",
		"stun://stun.altar.com.pl:3478",
	}
	output, err := inputLoaderStringListToModelExperimentTarget(input, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(input) != len(output) {
		t.Fatal("unexpected output length")
	}
	for idx := 0; idx < len(input); idx++ {
		if input[idx] != output[idx].Input() {
			t.Fatal("unexpected entry")
		}
		if output[idx].Category() != model.DefaultCategoryCode {
			t.Fatal("unexpected category")
		}
		if output[idx].Country() != model.DefaultCountryCode {
			t.Fatal("unexpected country")
		}
	}
}

func TestStringListToModelURLInfoWithInvalidInput(t *testing.T) {
	input := []string{
		"stun://stun.voip.blackberry.com:3478",
		"\t", // <- not a valid URL
		"stun://stun.altar.com.pl:3478",
	}
	output, err := inputLoaderStringListToModelExperimentTarget(input, nil)
	if err == nil || !strings.HasSuffix(err.Error(), "invalid control character in URL") {
		t.Fatal("no the error we expected", err)
	}
	if output != nil {
		t.Fatal("unexpected nil output")
	}
}

func TestStringListToModelURLInfoWithError(t *testing.T) {
	input := []string{
		"stun://stun.voip.blackberry.com:3478",
		"\t",
		"stun://stun.altar.com.pl:3478",
	}
	expected := errors.New("mocked error")
	output, err := inputLoaderStringListToModelExperimentTarget(input, expected)
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected", err)
	}
	if output != nil {
		t.Fatal("unexpected nil output")
	}
}
