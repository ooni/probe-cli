// Package oonitest contains code used for testing.
package oonitest

import (
	"context"
	"sync"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/cmd/ooniprobe/internal/config"
	"github.com/ooni/probe-cli/v3/cmd/ooniprobe/internal/ooni"
	"github.com/upper/db/v4"
)

// FakeOutput allows to fake the output package.
type FakeOutput struct {
	FakeSectionTitle []string
	mu               sync.Mutex
}

// SectionTitle writes the section title.
func (fo *FakeOutput) SectionTitle(s string) {
	fo.mu.Lock()
	defer fo.mu.Unlock()
	fo.FakeSectionTitle = append(fo.FakeSectionTitle, s)
}

// FakeProbeCLI fakes ooni.ProbeCLI
type FakeProbeCLI struct {
	FakeConfig         *config.Config
	FakeDB             db.Session
	FakeIsBatch        bool
	FakeHome           string
	FakeTempDir        string
	FakeProbeEnginePtr ooni.ProbeEngine
	FakeProbeEngineErr error
}

// Config implements ProbeCLI.Config
func (cli *FakeProbeCLI) Config() *config.Config {
	return cli.FakeConfig
}

// DB implements ProbeCLI.DB
func (cli *FakeProbeCLI) DB() db.Session {
	return cli.FakeDB
}

// IsBatch implements ProbeCLI.IsBatch
func (cli *FakeProbeCLI) IsBatch() bool {
	return cli.FakeIsBatch
}

// Home implements ProbeCLI.Home
func (cli *FakeProbeCLI) Home() string {
	return cli.FakeHome
}

// TempDir implements ProbeCLI.TempDir
func (cli *FakeProbeCLI) TempDir() string {
	return cli.FakeTempDir
}

// NewProbeEngine implements ProbeCLI.NewProbeEngine
func (cli *FakeProbeCLI) NewProbeEngine(ctx context.Context) (ooni.ProbeEngine, error) {
	return cli.FakeProbeEnginePtr, cli.FakeProbeEngineErr
}

var _ ooni.ProbeCLI = &FakeProbeCLI{}

// FakeProbeEngine fakes ooni.ProbeEngine
type FakeProbeEngine struct {
	FakeClose               error
	FakeMaybeLookupLocation error
	FakeProbeASNString      string
	FakeProbeCC             string
	FakeProbeIP             string
	FakeProbeNetworkName    string
}

// Close implements ProbeEngine.Close
func (eng *FakeProbeEngine) Close() error {
	return eng.FakeClose
}

// MaybeLookupLocation implements ProbeEngine.MaybeLookupLocation
func (eng *FakeProbeEngine) MaybeLookupLocation() error {
	return eng.FakeMaybeLookupLocation
}

// ProbeASNString implements ProbeEngine.ProbeASNString
func (eng *FakeProbeEngine) ProbeASNString() string {
	return eng.FakeProbeASNString
}

// ProbeCC implements ProbeEngine.ProbeCC
func (eng *FakeProbeEngine) ProbeCC() string {
	return eng.FakeProbeCC
}

// ProbeIP implements ProbeEngine.ProbeIP
func (eng *FakeProbeEngine) ProbeIP() string {
	return eng.FakeProbeIP
}

// ProbeNetworkName implements ProbeEngine.ProbeNetworkName
func (eng *FakeProbeEngine) ProbeNetworkName() string {
	return eng.FakeProbeNetworkName
}

var _ ooni.ProbeEngine = &FakeProbeEngine{}

// FakeLoggerHandler fakes apex.log.Handler.
type FakeLoggerHandler struct {
	FakeEntries []*log.Entry
	FakeErr     error
	mu          sync.Mutex
}

// HandleLog implements Handler.HandleLog.
func (handler *FakeLoggerHandler) HandleLog(entry *log.Entry) error {
	handler.mu.Lock()
	defer handler.mu.Unlock()
	handler.FakeEntries = append(handler.FakeEntries, entry)
	return handler.FakeErr
}

var _ log.Handler = &FakeLoggerHandler{}
