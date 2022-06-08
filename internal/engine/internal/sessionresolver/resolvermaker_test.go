package sessionresolver

import (
	"errors"
	"strings"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/bytecounter"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
)

func TestDefaultByteCounter(t *testing.T) {
	reso := &Resolver{}
	bc := reso.byteCounter()
	if bc == nil {
		t.Fatal("expected non-nil byte counter")
	}
}

func TestDefaultLogger(t *testing.T) {
	t.Run("when using a different logger", func(t *testing.T) {
		logger := &mocks.Logger{}
		reso := &Resolver{Logger: logger}
		lo := reso.logger()
		if lo != logger {
			t.Fatal("expected another logger here")
		}
	})

	t.Run("when no logger is set", func(t *testing.T) {
		reso := &Resolver{Logger: nil}
		lo := reso.logger()
		if lo != model.DiscardLogger {
			t.Fatal("expected another logger here")
		}
	})
}

func TestGetResolverHTTPSStandard(t *testing.T) {
	bc := bytecounter.New()
	URL := "https://dns.google"
	var closed bool
	re := &mocks.Resolver{
		MockCloseIdleConnections: func() {
			closed = true
		},
	}
	cmk := &fakeDNSClientMaker{reso: re}
	reso := &Resolver{dnsClientMaker: cmk, ByteCounter: bc}
	out, err := reso.getresolver(URL)
	if err != nil {
		t.Fatal(err)
	}
	if out != re {
		t.Fatal("not the result we expected")
	}
	o2, err := reso.getresolver(URL)
	if err != nil {
		t.Fatal(err)
	}
	if out != o2 {
		t.Fatal("not the result we expected")
	}
	reso.closeall()
	if closed != true {
		t.Fatal("was not closed")
	}
	if cmk.savedURL != URL {
		t.Fatal("not the URL we expected")
	}
	if cmk.savedConfig.ByteCounter != bc {
		t.Fatal("unexpected ByteCounter")
	}
	if cmk.savedConfig.BogonIsError != true {
		t.Fatal("unexpected BogonIsError")
	}
	if cmk.savedConfig.HTTP3Enabled != false {
		t.Fatal("unexpected HTTP3Enabled")
	}
	if cmk.savedConfig.Logger != model.DiscardLogger {
		t.Fatal("unexpected Log")
	}
}

func TestGetResolverHTTP3(t *testing.T) {
	bc := bytecounter.New()
	URL := "http3://dns.google"
	var closed bool
	re := &mocks.Resolver{
		MockCloseIdleConnections: func() {
			closed = true
		},
	}
	cmk := &fakeDNSClientMaker{reso: re}
	reso := &Resolver{dnsClientMaker: cmk, ByteCounter: bc}
	out, err := reso.getresolver(URL)
	if err != nil {
		t.Fatal(err)
	}
	if out != re {
		t.Fatal("not the result we expected")
	}
	o2, err := reso.getresolver(URL)
	if err != nil {
		t.Fatal(err)
	}
	if out != o2 {
		t.Fatal("not the result we expected")
	}
	reso.closeall()
	if closed != true {
		t.Fatal("was not closed")
	}
	if cmk.savedURL != strings.Replace(URL, "http3://", "https://", 1) {
		t.Fatal("not the URL we expected")
	}
	if cmk.savedConfig.ByteCounter != bc {
		t.Fatal("unexpected ByteCounter")
	}
	if cmk.savedConfig.BogonIsError != true {
		t.Fatal("unexpected BogonIsError")
	}
	if cmk.savedConfig.HTTP3Enabled != true {
		t.Fatal("unexpected HTTP3Enabled")
	}
	if cmk.savedConfig.Logger != model.DiscardLogger {
		t.Fatal("unexpected Log")
	}
}

func TestGetResolverInvalidURL(t *testing.T) {
	bc := bytecounter.New()
	URL := "http3://dns.google"
	errMocked := errors.New("mocked error")
	cmk := &fakeDNSClientMaker{err: errMocked}
	reso := &Resolver{dnsClientMaker: cmk, ByteCounter: bc}
	out, err := reso.getresolver(URL)
	if !errors.Is(err, errMocked) {
		t.Fatal("not the error we expected", err)
	}
	if out != nil {
		t.Fatal("not the result we expected")
	}
}
