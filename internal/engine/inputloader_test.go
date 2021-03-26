package engine

import (
	"context"
	"errors"
	"io"
	"os"
	"syscall"
	"testing"

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

type InputLoaderBrokenSession struct {
	OrchestraClient model.ExperimentOrchestraClient
	Error           error
}

func (InputLoaderBrokenSession) MaybeLookupLocationContext(ctx context.Context) error {
	return nil
}

func (ilbs InputLoaderBrokenSession) NewOrchestraClient(ctx context.Context) (model.ExperimentOrchestraClient, error) {
	if ilbs.OrchestraClient != nil {
		return ilbs.OrchestraClient, nil
	}
	return nil, io.EOF
}

func (InputLoaderBrokenSession) ProbeCC() string {
	return "IT"
}

func TestInputLoaderNewOrchestraClientFailure(t *testing.T) {
	il := inputLoader{}
	lrc := inputLoaderLoadRemoteConfig{
		ctx:     context.Background(),
		session: InputLoaderBrokenSession{},
	}
	out, err := il.loadRemote(lrc)
	if !errors.Is(err, io.EOF) {
		t.Fatalf("not the error we expected: %+v", err)
	}
	if out != nil {
		t.Fatal("expected nil output here")
	}
}

type InputLoaderBrokenOrchestraClient struct{}

func (InputLoaderBrokenOrchestraClient) CheckIn(ctx context.Context, config model.CheckInConfig) (*model.CheckInInfo, error) {
	return nil, io.EOF
}

func (InputLoaderBrokenOrchestraClient) FetchPsiphonConfig(ctx context.Context) ([]byte, error) {
	return nil, io.EOF
}

func (InputLoaderBrokenOrchestraClient) FetchTorTargets(ctx context.Context, cc string) (map[string]model.TorTarget, error) {
	return nil, io.EOF
}

func (InputLoaderBrokenOrchestraClient) FetchURLList(ctx context.Context, config model.URLListConfig) ([]model.URLInfo, error) {
	return nil, io.EOF
}

func TestInputLoaderFetchURLListFailure(t *testing.T) {
	il := inputLoader{}
	lrc := inputLoaderLoadRemoteConfig{
		ctx: context.Background(),
		session: InputLoaderBrokenSession{
			OrchestraClient: InputLoaderBrokenOrchestraClient{},
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
