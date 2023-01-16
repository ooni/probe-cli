package runtimex

import (
	"errors"
	"runtime/debug"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestPanicOnError(t *testing.T) {
	badfunc := func(in error) (out error) {
		defer func() {
			out = recover().(error)
		}()
		PanicOnError(in, "we expect this assertion to fail")
		return
	}

	t.Run("error is nil", func(t *testing.T) {
		PanicOnError(nil, "this assertion should not fail")
	})

	t.Run("error is not nil", func(t *testing.T) {
		expected := errors.New("mocked error")
		if !errors.Is(badfunc(expected), expected) {
			t.Fatal("not the error we expected")
		}
	})
}

func TestAssert(t *testing.T) {
	badfunc := func(in bool, message string) (out error) {
		defer func() {
			out = recover().(error)
		}()
		Assert(in, message)
		return
	}

	t.Run("assertion is true", func(t *testing.T) {
		Assert(true, "this assertion should not fail")
	})

	t.Run("assertion is false", func(t *testing.T) {
		message := "mocked error"
		err := badfunc(false, message)
		if err == nil || err.Error() != message {
			t.Fatal("not the error we expected", err)
		}
	})
}

func TestPanicIfTrue(t *testing.T) {
	badfunc := func(in bool, message string) (out error) {
		defer func() {
			out = recover().(error)
		}()
		PanicIfTrue(in, message)
		return
	}

	t.Run("assertion is false", func(t *testing.T) {
		PanicIfTrue(false, "this assertion should not fail")
	})

	t.Run("assertion is true", func(t *testing.T) {
		message := "mocked error"
		err := badfunc(true, message)
		if err == nil || err.Error() != message {
			t.Fatal("not the error we expected", err)
		}
	})
}

func TestPanicIfNil(t *testing.T) {
	badfunc := func(in interface{}, message string) (out error) {
		defer func() {
			out = recover().(error)
		}()
		PanicIfNil(in, message)
		return
	}

	t.Run("value is not nil", func(t *testing.T) {
		PanicIfNil(false, "this assertion should not fail")
	})

	t.Run("value is nil", func(t *testing.T) {
		message := "mocked error"
		err := badfunc(nil, message)
		if err == nil || err.Error() != message {
			t.Fatal("not the error we expected", err)
		}
	})
}

func TestBuildInfoRecord_setall(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value string
		want  *BuildInfoRecord
	}{{
		name:  "for VcsModified",
		key:   "vcs.modified",
		value: "ABC",
		want: &BuildInfoRecord{
			VcsModified: "ABC",
		},
	}, {
		name:  "for VcsRevision",
		key:   "vcs.revision",
		value: "ABC",
		want: &BuildInfoRecord{
			VcsRevision: "ABC",
		},
	}, {
		name:  "for VcsTime",
		key:   "vcs.time",
		value: "ABC",
		want: &BuildInfoRecord{
			VcsTime: "ABC",
		},
	}, {
		name:  "for VcsTool",
		key:   "vcs",
		value: "git",
		want: &BuildInfoRecord{
			VcsTool: "git",
		},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bir := &BuildInfoRecord{
				GoVersion:   "",
				VcsModified: "",
				VcsRevision: "",
				VcsTime:     "",
				VcsTool:     "",
			}
			bir.setall([]debug.BuildSetting{{Key: tt.key, Value: tt.value}})
			if diff := cmp.Diff(tt.want, bir); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
