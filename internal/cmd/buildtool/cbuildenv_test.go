package main

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestCBuildEnv(t *testing.T) {
	t.Run("we can correctly merge build flags", func(t *testing.T) {
		global := &cBuildEnv{
			CFLAGS:   []string{"a", "b", "c"},
			CXXFLAGS: []string{"A", "B", "C"},
			LDFLAGS:  []string{"1", "2", "3"},
		}
		local := &cBuildEnv{
			CFLAGS:   []string{"d", "e"},
			CXXFLAGS: []string{"D", "E"},
			LDFLAGS:  []string{"4", "5"},
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

	t.Run("we do nothing with empty variables", func(t *testing.T) {
		global := &cBuildEnv{
			CFLAGS:   []string{},
			CXXFLAGS: []string{},
			LDFLAGS:  []string{},
		}
		local := &cBuildEnv{
			CFLAGS:   []string{},
			CXXFLAGS: []string{},
			LDFLAGS:  []string{},
		}
		envp := cBuildExportEnviron(global, local)
		var expected []string
		if diff := cmp.Diff(expected, envp.V); diff != "" {
			t.Fatal(diff)
		}
	})
}
