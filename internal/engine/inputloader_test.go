package engine

import (
	"context"
	"errors"
	"io"
	"os"
	"syscall"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/engine/internal/fsx"
	"github.com/ooni/probe-cli/v3/internal/engine/model"
)

type InputLoaderBrokenFS struct{}

func (InputLoaderBrokenFS) Open(filepath string) (fsx.File, error) {
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
	il := inputLoader{}
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
	il := inputLoader{}
	lrc := inputLoaderLoadRemoteConfig{
		ctx: context.Background(),
		session: &InputLoaderMockableSession{
			Error: io.EOF,
		},
	}
	out, err := il.loadRemote(lrc)
	if !errors.Is(err, io.EOF) {
		t.Fatalf("not the error we expected: %+v", err)
	}
	if out != nil {
		t.Fatal("expected nil output here")
	}
}

func TestInputLoaderCheckInSuccessWithNilWebConnectivity(t *testing.T) {
	il := inputLoader{}
	lrc := inputLoaderLoadRemoteConfig{
		ctx: context.Background(),
		session: &InputLoaderMockableSession{
			Output: &model.CheckInInfo{},
		},
	}
	out, err := il.loadRemote(lrc)
	if !errors.Is(err, ErrNoURLsReturned) {
		t.Fatalf("not the error we expected: %+v", err)
	}
	if out != nil {
		t.Fatal("expected nil output here")
	}
}

func TestInputLoaderCheckInSuccessWithNoURLs(t *testing.T) {
	il := inputLoader{}
	lrc := inputLoaderLoadRemoteConfig{
		ctx: context.Background(),
		session: &InputLoaderMockableSession{
			Output: &model.CheckInInfo{
				WebConnectivity: &model.CheckInInfoWebConnectivity{},
			},
		},
	}
	out, err := il.loadRemote(lrc)
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
	il := inputLoader{}
	lrc := inputLoaderLoadRemoteConfig{
		ctx: context.Background(),
		session: &InputLoaderMockableSession{
			Output: &model.CheckInInfo{
				WebConnectivity: &model.CheckInInfoWebConnectivity{
					URLs: expect,
				},
			},
		},
	}
	out, err := il.loadRemote(lrc)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(expect, out); diff != "" {
		t.Fatal(diff)
	}
}
