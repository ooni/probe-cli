package kvstore2dir

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

type booleanStatBuf bool

var _ statBuf = booleanStatBuf(true)

// IsDir implements statBuf.
func (v booleanStatBuf) IsDir() bool {
	return bool(v)
}

func TestMove(t *testing.T) {
	// testcase is a test case implemented by this function
	type testcase struct {
		name     string
		osStat   func(name string) (statBuf, error)
		osRename func(oldpath string, newpath string) error
		expect   error
	}

	cases := []testcase{{
		name: "when we cannot stat kvstore2",
		osStat: func(name string) (statBuf, error) {
			return nil, io.EOF
		},
		osRename: func(oldpath string, newpath string) error {
			panic("should not be called")
		},
		expect: nil,
	}, {
		name: "when kvstore2 is not a directory",
		osStat: func(name string) (statBuf, error) {
			return booleanStatBuf(false), nil
		},
		osRename: func(oldpath string, newpath string) error {
			panic("should not be called")
		},
		expect: nil,
	}, {
		name: "when we can find kvstore2 as a dir and engine",
		osStat: func(name string) (statBuf, error) {
			if name == filepath.Join("xo", "kvstore2") {
				return booleanStatBuf(true), nil
			}
			return booleanStatBuf(true), nil
		},
		osRename: func(oldpath string, newpath string) error {
			panic("should not be called")
		},
		expect: nil,
	}, {
		name: "when we can find kvstore2 as a dir without engine",
		osStat: func(name string) (statBuf, error) {
			if name == filepath.Join("xo", "kvstore2") {
				return booleanStatBuf(true), nil
			}
			return nil, io.EOF
		},
		osRename: func(oldpath string, newpath string) error {
			return nil
		},
		expect: nil,
	}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// override and restore functions
			osStat = tc.osStat
			osRename = tc.osRename
			defer func() {
				osStat = simplifiedStat
				osRename = os.Rename
			}()

			// invoke Move
			err := Move("xo")

			// check the result
			if !errors.Is(err, tc.expect) {
				t.Fatal("expected", tc.expect, "got", err)
			}
		})
	}
}

func TestSimplifiedStat(t *testing.T) {
	buf, err := simplifiedStat("kvstore2dir.go")
	if err != nil {
		t.Fatal(err)
	}
	if buf.IsDir() {
		t.Fatal("expected not dir")
	}
}
