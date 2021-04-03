package engine

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"syscall"
	"testing"

	"github.com/apex/log"
	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/engine/kvstore"
	"github.com/ooni/probe-cli/v3/internal/engine/model"
)

func TestInputLoaderInputNoneWithStaticInputs(t *testing.T) {
	il := &InputLoader{
		StaticInputs: []string{"https://www.google.com/"},
		InputPolicy:  InputNone,
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
		InputPolicy: InputNone,
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
		InputPolicy: InputNone,
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
		InputPolicy: InputNone,
	}
	ctx := context.Background()
	out, err := il.Load(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0].URL != "" {
		t.Fatal("not the output we expected")
	}
}

func TestInputLoaderInputOptionalWithNoInput(t *testing.T) {
	il := &InputLoader{
		InputPolicy: InputOptional,
	}
	ctx := context.Background()
	out, err := il.Load(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0].URL != "" {
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
		InputPolicy: InputOptional,
	}
	ctx := context.Background()
	out, err := il.Load(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 5 {
		t.Fatal("not the output length we expected")
	}
	expect := []model.URLInfo{
		{URL: "https://www.google.com/"},
		{URL: "https://www.x.org/"},
		{URL: "https://www.slashdot.org/"},
		{URL: "https://abc.xyz/"},
		{URL: "https://run.ooni.io/"},
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
		InputPolicy: InputOptional,
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
		InputPolicy: InputStrictlyRequired,
	}
	ctx := context.Background()
	out, err := il.Load(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 5 {
		t.Fatal("not the output length we expected")
	}
	expect := []model.URLInfo{
		{URL: "https://www.google.com/"},
		{URL: "https://www.x.org/"},
		{URL: "https://www.slashdot.org/"},
		{URL: "https://abc.xyz/"},
		{URL: "https://run.ooni.io/"},
	}
	if diff := cmp.Diff(out, expect); diff != "" {
		t.Fatal(diff)
	}
}

func TestInputLoaderInputStrictlyRequiredWithoutInput(t *testing.T) {
	il := &InputLoader{
		InputPolicy: InputStrictlyRequired,
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
		InputPolicy: InputStrictlyRequired,
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

func TestInputLoaderInputOrQueryBackendWithInput(t *testing.T) {
	il := &InputLoader{
		StaticInputs: []string{"https://www.google.com/"},
		SourceFiles: []string{
			"testdata/inputloader1.txt",
			"testdata/inputloader2.txt",
		},
		InputPolicy: InputOrQueryBackend,
	}
	ctx := context.Background()
	out, err := il.Load(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 5 {
		t.Fatal("not the output length we expected")
	}
	expect := []model.URLInfo{
		{URL: "https://www.google.com/"},
		{URL: "https://www.x.org/"},
		{URL: "https://www.slashdot.org/"},
		{URL: "https://abc.xyz/"},
		{URL: "https://run.ooni.io/"},
	}
	if diff := cmp.Diff(out, expect); diff != "" {
		t.Fatal(diff)
	}
}

func TestInputLoaderInputOrQueryBackendWithNoInputAndCancelledContext(t *testing.T) {
	sess, err := NewSession(context.Background(), SessionConfig{
		KVStore:         kvstore.NewMemoryKeyValueStore(),
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
		InputPolicy: InputOrQueryBackend,
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
		InputPolicy: InputOrQueryBackend,
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
	Output *model.CheckInInfo

	// Error is the error to be returned by CheckIn. It
	// should be nil when Output is not-nil.
	Error error
}

// CheckIn implements InputLoaderSession.CheckIn.
func (sess *InputLoaderMockableSession) CheckIn(
	ctx context.Context, config *model.CheckInConfig) (*model.CheckInInfo, error) {
	if sess.Output == nil && sess.Error == nil {
		return nil, errors.New("both Output and Error are nil")
	}
	return sess.Output, sess.Error
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
			Output: &model.CheckInInfo{},
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
			Output: &model.CheckInInfo{
				WebConnectivity: &model.CheckInInfoWebConnectivity{},
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
	expect := []model.URLInfo{{
		CategoryCode: "NEWS",
		CountryCode:  "IT",
		URL:          "https://repubblica.it",
	}, {
		CategoryCode: "NEWS",
		CountryCode:  "IT",
		URL:          "https://corriere.it",
	}}
	il := &InputLoader{
		Session: &InputLoaderMockableSession{
			Output: &model.CheckInInfo{
				WebConnectivity: &model.CheckInInfoWebConnectivity{
					URLs: expect,
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
