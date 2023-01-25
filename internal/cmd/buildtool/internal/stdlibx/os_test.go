package stdlibx

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestRegularFileExists(t *testing.T) {
	stdlib := NewStdlib()

	t.Run("for an existing regular file", func(t *testing.T) {
		filename := filepath.Join("testdata", ".gitignore")
		if !stdlib.RegularFileExists(filename) {
			t.Fatal("file should exist")
		}
	})

	t.Run("for an existing directory", func(t *testing.T) {
		if stdlib.RegularFileExists("testdata") {
			t.Fatal("should not detect existing directory as regular file")
		}
	})

	t.Run("for a nonexistent path", func(t *testing.T) {
		filename := filepath.Join("testdata", "nonexistent")
		if stdlib.RegularFileExists(filename) {
			t.Fatal("file should not exist")
		}
	})
}

type testExiter struct {
	err error
}

func (te *testExiter) Exit(code int) {
	panic(te.err)
}

func TestMustReadFileFirstLine(t *testing.T) {
	t.Run("for an existing file", func(t *testing.T) {
		stdlib := NewStdlib()
		filename := filepath.Join("testdata", ".gitignore")
		data := stdlib.MustReadFileFirstLine(filename)
		if data != "*" {
			t.Fatal("unexpected data", data)
		}
	})

	t.Run("for a nonexisting file", func(t *testing.T) {
		expected := errors.New("mocked error")
		var got error
		func() {
			defer func() {
				if r := recover(); r != nil {
					got = r.(error)
				}
			}()
			stdlib := NewStdlib().(*stdlib)
			stdlib.exiter = &testExiter{
				err: expected,
			}
			filename := filepath.Join("testdata", "nonexistent")
			_ = stdlib.MustReadFileFirstLine(filename)
		}()
		if expected != got {
			t.Fatal("did not call exit")
		}
	})
}

func testCompareTwoFiles(source, dest string) error {
	data1, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	data2, err := os.ReadFile(dest)
	if err != nil {
		return err
	}
	if diff := cmp.Diff(data1, data2); diff != "" {
		return errors.New(diff)
	}
	return nil
}

func TestCopyFile(t *testing.T) {
	stdlib := NewStdlib()

	t.Run("for an existing file", func(t *testing.T) {
		source := "exec.go"
		dest := filepath.Join("testdata", "exec.go")
		err := stdlib.CopyFile(source, dest)
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(dest)
		if err := testCompareTwoFiles(source, dest); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("for a nonexistent file", func(t *testing.T) {
		source := filepath.Join("testdata", "nonexistent")
		dest := filepath.Join("testdata", "antani")
		err := stdlib.CopyFile(source, dest)
		if !errors.Is(err, syscall.ENOENT) {
			t.Fatal("unexpected error", err)
		}
	})
}

func TestMustWriteFile(t *testing.T) {
	expectedContent := []byte("something\nhas multiple\nlines\n")

	t.Run("in case of success", func(t *testing.T) {
		stdlib := NewStdlib()
		dest := filepath.Join("testdata", "antani")
		defer os.Remove(dest)
		stdlib.MustWriteFile(dest, expectedContent, 0600)
		data, err := os.ReadFile(dest)
		if err != nil {
			t.Fatal(err)
		}
		if diff := cmp.Diff(expectedContent, data); diff != "" {
			t.Fatal(diff)
		}
	})

	t.Run("in case of failure", func(t *testing.T) {
		expected := errors.New("mocked error")
		var got error
		func() {
			defer func() {
				if r := recover(); r != nil {
					got = r.(error)
				}
			}()
			stdlib := NewStdlib().(*stdlib)
			stdlib.exiter = &testExiter{
				err: expected,
			}
			filename := filepath.Join("testdata", "nonexistent", "nonexistent")
			stdlib.MustWriteFile(filename, expectedContent, 0600)
		}()
		if expected != got {
			t.Fatal("did not call exit")
		}
	})
}
