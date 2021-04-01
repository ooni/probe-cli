package assetsdir

import (
	"errors"
	"strings"
	"testing"
)

func TestCleanupNormalUsage(t *testing.T) {
	result, err := Cleanup("testdata")
	if err != nil {
		t.Fatal(err)
	}
	// we expect a bunch of ENOENT because the directory does not exist.
	isExpectedErr := func(err error) bool {
		return err != nil && strings.HasSuffix(err.Error(), "no such file or directory")
	}
	if !isExpectedErr(result.ASNDatabaseErr) {
		t.Fatal("unexpected error", result.ASNDatabaseErr)
	}
	if !isExpectedErr(result.CABundleErr) {
		t.Fatal("unexpected error", result.CABundleErr)
	}
	if !isExpectedErr(result.CountryDatabaseErr) {
		t.Fatal("unexpected error", result.CountryDatabaseErr)
	}
	if !isExpectedErr(result.RmdirErr) {
		t.Fatal("unexpected error", result.RmdirErr)
	}
}

func TestCleanupWithEmptyInput(t *testing.T) {
	result, err := Cleanup("")
	if !errors.Is(err, ErrEmptyDir) {
		t.Fatal("unexpected error", err)
	}
	if result != nil {
		t.Fatal("expected nil result")
	}
}
