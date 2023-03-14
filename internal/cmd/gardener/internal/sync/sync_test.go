package sync_test

import (
	"path/filepath"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/cmd/gardener/internal/sync"
)

func TestWorkingAsIntended(t *testing.T) {
	sc := &sync.Subcommand{
		RepositoryDir: filepath.Join("testdata", "repo"),
	}
	sc.Main()
}
