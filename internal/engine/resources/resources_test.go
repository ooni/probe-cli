package resources_test

import (
	"compress/gzip"
	"context"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/engine/resources"
)

func TestEnsureMkdirAllFailure(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	expected := errors.New("mocked error")
	client := resources.Client{
		HTTPClient: http.DefaultClient,
		Logger:     log.Log,
		OSMkdirAll: func(string, os.FileMode) error {
			return expected
		},
		UserAgent: "ooniprobe-engine/0.1.0",
		WorkDir:   "/foobar",
	}
	err := client.Ensure(context.Background())
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected")
	}
}

func TestEnsure(t *testing.T) {
	tempdir, err := ioutil.TempDir("", "ooniprobe-engine-resources-test")
	if err != nil {
		t.Fatal(err)
	}
	client := resources.Client{
		HTTPClient: http.DefaultClient,
		Logger:     log.Log,
		UserAgent:  "ooniprobe-engine/0.1.0",
		WorkDir:    tempdir,
	}
	err = client.Ensure(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	// the second round should be idempotent
	err = client.Ensure(context.Background())
	if err != nil {
		t.Fatal(err)
	}
}

func TestEnsureFailure(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	tempdir, err := ioutil.TempDir("", "ooniprobe-engine-resources-test")
	if err != nil {
		t.Fatal(err)
	}
	client := resources.Client{
		HTTPClient: http.DefaultClient,
		Logger:     log.Log,
		UserAgent:  "ooniprobe-engine/0.1.0",
		WorkDir:    tempdir,
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err = client.Ensure(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatal("not the error we expected")
	}
}

func TestEnsureFailAllComparisons(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	tempdir, err := ioutil.TempDir("", "ooniprobe-engine-resources-test")
	if err != nil {
		t.Fatal(err)
	}
	client := resources.Client{
		HTTPClient: http.DefaultClient,
		Logger:     log.Log,
		UserAgent:  "ooniprobe-engine/0.1.0",
		WorkDir:    tempdir,
	}
	// run once to download the resource once
	err = client.EnsureForSingleResource(
		context.Background(), "ca-bundle.pem", resources.ResourceInfo{
			URLPath:  "/ooni/probe-assets/releases/download/20190822135402/ca-bundle.pem.gz",
			GzSHA256: "d5a6aa2290ee18b09cc4fb479e2577ed5ae66c253870ba09776803a5396ea3ab",
			SHA256:   "cb2eca3fbfa232c9e3874e3852d43b33589f27face98eef10242a853d83a437a",
		}, func(left, right string) bool {
			return left == right
		},
		gzip.NewReader, ioutil.ReadAll,
	)
	if err != nil {
		t.Fatal(err)
	}
	// re-run with broken comparison operator so that we should
	// first redownload and then fail for invalid SHA256.
	err = client.EnsureForSingleResource(
		context.Background(), "ca-bundle.pem", resources.ResourceInfo{
			URLPath:  "/ooni/probe-assets/releases/download/20190822135402/ca-bundle.pem.gz",
			GzSHA256: "d5a6aa2290ee18b09cc4fb479e2577ed5ae66c253870ba09776803a5396ea3ab",
			SHA256:   "cb2eca3fbfa232c9e3874e3852d43b33589f27face98eef10242a853d83a437a",
		}, func(left, right string) bool {
			return false // comparison for equality always fails
		},
		gzip.NewReader, ioutil.ReadAll,
	)
	if err == nil || !strings.HasSuffix(err.Error(), "sha256 mismatch") {
		t.Fatal("not the error we expected")
	}
}

func TestEnsureFailGzipNewReader(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	tempdir, err := ioutil.TempDir("", "ooniprobe-engine-resources-test")
	if err != nil {
		t.Fatal(err)
	}
	client := resources.Client{
		HTTPClient: http.DefaultClient,
		Logger:     log.Log,
		UserAgent:  "ooniprobe-engine/0.1.0",
		WorkDir:    tempdir,
	}
	expected := errors.New("mocked error")
	err = client.EnsureForSingleResource(
		context.Background(), "ca-bundle.pem", resources.ResourceInfo{
			URLPath:  "/ooni/probe-assets/releases/download/20190822135402/ca-bundle.pem.gz",
			GzSHA256: "d5a6aa2290ee18b09cc4fb479e2577ed5ae66c253870ba09776803a5396ea3ab",
			SHA256:   "cb2eca3fbfa232c9e3874e3852d43b33589f27face98eef10242a853d83a437a",
		}, func(left, right string) bool {
			return left == right
		},
		func(r io.Reader) (*gzip.Reader, error) {
			return nil, expected
		},
		ioutil.ReadAll,
	)
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected")
	}
}

func TestEnsureFailIoUtilReadAll(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	tempdir, err := ioutil.TempDir("", "ooniprobe-engine-resources-test")
	if err != nil {
		t.Fatal(err)
	}
	client := resources.Client{
		HTTPClient: http.DefaultClient,
		Logger:     log.Log,
		UserAgent:  "ooniprobe-engine/0.1.0",
		WorkDir:    tempdir,
	}
	expected := errors.New("mocked error")
	err = client.EnsureForSingleResource(
		context.Background(), "ca-bundle.pem", resources.ResourceInfo{
			URLPath:  "/ooni/probe-assets/releases/download/20190822135402/ca-bundle.pem.gz",
			GzSHA256: "d5a6aa2290ee18b09cc4fb479e2577ed5ae66c253870ba09776803a5396ea3ab",
			SHA256:   "cb2eca3fbfa232c9e3874e3852d43b33589f27face98eef10242a853d83a437a",
		}, func(left, right string) bool {
			return left == right
		},
		gzip.NewReader, func(r io.Reader) ([]byte, error) {
			return nil, expected
		},
	)
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected")
	}
}
