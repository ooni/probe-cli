package flagx_test

// The code in this file is adapted from github.com/m-lab/go and more
// specifically from <https://git.io/JJ8UA>. This file is licensed under
// version 2.0 of the Apache License <https://git.io/JJ8Ux>.

import (
	"flag"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/engine/cmd/jafar/flagx"
)

func TestStringArray(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expt     flagx.StringArray
		repr     string
		contains string
		wantErr  bool
	}{
		{
			name:     "okay",
			args:     []string{"a", "b"},
			expt:     flagx.StringArray{"a", "b"},
			repr:     `[]string{"a", "b"}`,
			contains: "b",
		},
		{
			name:     "okay-split-commas",
			args:     []string{"a", "b", "c,d"},
			expt:     flagx.StringArray{"a", "b", "c", "d"},
			repr:     `[]string{"a", "b", "c", "d"}`,
			contains: "d",
		},
		{
			name:     "empty",
			args:     []string{},
			expt:     flagx.StringArray{},
			repr:     `[]string{}`,
			contains: "a",
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sa := &flagx.StringArray{}
			for i := range tt.args {
				if err := sa.Set(tt.args[i]); err != nil {
					t.Errorf("StringArray.Set() error = %v, want nil", err)
				}
			}
			v := (sa.Get().(flagx.StringArray))
			if diff := cmp.Diff(v, tt.expt); diff != "" {
				t.Errorf("StringArray.Get() unexpected differences %v", diff)
			}
			if tt.repr != sa.String() {
				t.Errorf("StringArray.String() want = %q, got %q", tt.repr, sa.String())
			}
			if sa.Contains(tt.contains) == tt.wantErr {
				t.Errorf("StringArray.Contains() want = %q, got %t", tt.repr, sa.Contains(tt.contains))
			}
		})
	}
}

// Successful compilation of this function means that StringArray implements the
// flag.Getter interface. The function need not be called.
func assertFlagGetterStringArray(b flagx.StringArray) {
	func(in flag.Getter) {}(&b)
}
