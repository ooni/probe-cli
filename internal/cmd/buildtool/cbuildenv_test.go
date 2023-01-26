package main

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestCBuildEnv(t *testing.T) {
	t.Run("we can correctly merge build flags", func(t *testing.T) {
		global := &cBuildEnv{
			cflags:   []string{"a", "b", "c"},
			cxxflags: []string{"A", "B", "C"},
			ldflags:  []string{"1", "2", "3"},
		}
		local := &cBuildEnv{
			cflags:   []string{"d", "e"},
			cxxflags: []string{"D", "E"},
			ldflags:  []string{"4", "5"},
		}
		envp := cBuildExportEnviron(global, local)
		expected := []string{
			"CFLAGS=a b c d e",
			"CXXFLAGS=A B C D E",
			"LDFLAGS=1 2 3 4 5",
		}
		if diff := cmp.Diff(expected, envp.V); diff != "" {
			t.Fatal(diff)
		}
	})
}
