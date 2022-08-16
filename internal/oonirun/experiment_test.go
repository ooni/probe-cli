package oonirun

import (
	"context"
	"os"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine"
	"github.com/ooni/probe-cli/v3/internal/kvstore"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/version"
)

// TODO(bassosimone): it would be cool to write unit tests. However, to do that
// we need to ~redesign the engine package for unit-testability.

func newSession(ctx context.Context, t *testing.T) *engine.Session {
	config := engine.SessionConfig{
		AvailableProbeServices: []model.OOAPIService{},
		KVStore:                &kvstore.Memory{},
		Logger:                 model.DiscardLogger,
		ProxyURL:               nil,
		SoftwareName:           "miniooni",
		SoftwareVersion:        version.Version,
		TempDir:                os.TempDir(),
		TorArgs:                []string{},
		TorBinary:              "",
		TunnelDir:              "",
	}
	sess, err := engine.NewSession(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	return sess
}

func TestExperimentRunWithExample(t *testing.T) {
	ctx := context.Background()
	desc := &Experiment{
		Annotations: map[string]string{
			"platform": "linux",
		},
		ExtraOptions: map[string]any{
			"SleepTime": int64(10 * time.Millisecond),
		},
		Inputs:         []string{},
		InputFilePaths: []string{},
		MaxRuntime:     0,
		Name:           "example",
		NoCollector:    true,
		NoJSON:         true,
		Random:         false,
		ReportFile:     "",
		Session:        newSession(ctx, t),
	}
	if err := desc.Run(ctx); err != nil {
		t.Fatal(err)
	}
}

func Test_experimentOptionsToStringList(t *testing.T) {
	type args struct {
		options map[string]any
	}
	tests := []struct {
		name    string
		args    args
		wantOut []string
	}{
		{
			name: "happy path: a map with three entries returns three items",
			args: args{
				map[string]any{
					"foo":  1,
					"bar":  2,
					"baaz": 3,
				},
			},
			wantOut: []string{"baaz=3", "bar=2", "foo=1"},
		},
		{
			name: "an option beginning with `Safe` is skipped from the output",
			args: args{
				map[string]any{
					"foo":     1,
					"Safefoo": 42,
				},
			},
			wantOut: []string{"foo=1"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOut := experimentOptionsToStringList(tt.args.options)
			sort.Strings(gotOut)
			if !reflect.DeepEqual(gotOut, tt.wantOut) {
				t.Errorf("experimentOptionsToStringList() = %v, want %v", gotOut, tt.wantOut)
			}
		})
	}
}
