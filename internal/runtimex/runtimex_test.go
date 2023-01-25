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

func TestTry(t *testing.T) {
	t.Run("Try0", func(t *testing.T) {
		t.Run("on success", func(t *testing.T) {
			Try0(nil)
		})

		t.Run("on failure", func(t *testing.T) {
			expected := errors.New("mocked error")
			var got error
			func() {
				defer func() {
					if r := recover(); r != nil {
						got = r.(error)
					}
				}()
				Try0(expected)
			}()
			if !errors.Is(got, expected) {
				t.Fatal("unexpected error")
			}
		})
	})

	t.Run("Try1", func(t *testing.T) {
		t.Run("on success", func(t *testing.T) {
			v1 := Try1(17, nil)
			if v1 != 17 {
				t.Fatal("unexpected value")
			}
		})

		t.Run("on failure", func(t *testing.T) {
			expected := errors.New("mocked error")
			var got error
			func() {
				defer func() {
					if r := recover(); r != nil {
						got = r.(error)
					}
				}()
				Try1(17, expected)
			}()
			if !errors.Is(got, expected) {
				t.Fatal("unexpected error")
			}
		})
	})

	t.Run("Try2", func(t *testing.T) {
		t.Run("on success", func(t *testing.T) {
			v1, v2 := Try2(17, true, nil)
			if v1 != 17 || !v2 {
				t.Fatal("unexpected value")
			}
		})

		t.Run("on failure", func(t *testing.T) {
			expected := errors.New("mocked error")
			var got error
			func() {
				defer func() {
					if r := recover(); r != nil {
						got = r.(error)
					}
				}()
				Try2(17, true, expected)
			}()
			if !errors.Is(got, expected) {
				t.Fatal("unexpected error")
			}
		})
	})

	t.Run("Try3", func(t *testing.T) {
		t.Run("on success", func(t *testing.T) {
			v1, v2, v3 := Try3(17, true, 44.0, nil)
			if v1 != 17 || !v2 || v3 != 44.0 {
				t.Fatal("unexpected value")
			}
		})

		t.Run("on failure", func(t *testing.T) {
			expected := errors.New("mocked error")
			var got error
			func() {
				defer func() {
					if r := recover(); r != nil {
						got = r.(error)
					}
				}()
				Try3(17, true, 44.0, expected)
			}()
			if !errors.Is(got, expected) {
				t.Fatal("unexpected error")
			}
		})
	})
}
