package sessionresolver

import (
	"strings"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/bytecounter"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
)

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
	var (
		savedURL string
		savedH3  bool
	)
	reso := &Resolver{
		ByteCounter: bc,
		newChildResolverFn: func(h3 bool, URL string) (model.Resolver, error) {
			savedURL = URL
			savedH3 = h3
			return re, nil
		},
	}
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
	if savedURL != URL {
		t.Fatal("not the URL we expected")
	}
	if savedH3 {
		t.Fatal("expected false")
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
	var (
		savedURL string
		savedH3  bool
	)
	reso := &Resolver{
		ByteCounter: bc,
		newChildResolverFn: func(h3 bool, URL string) (model.Resolver, error) {
			savedURL = URL
			savedH3 = h3
			return re, nil
		},
	}
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
	if savedURL != strings.Replace(URL, "http3://", "https://", 1) {
		t.Fatal("not the URL we expected")
	}
	if !savedH3 {
		t.Fatal("expected true")
	}
}
