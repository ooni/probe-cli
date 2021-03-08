package httptransport

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"testing"
)

func composeWithEOFError(msg string) error {
	return fmt.Errorf("%w: %s", io.EOF, msg)
}

func TestIgnoreExpectedEOFWithNoError(t *testing.T) {
	if err := ignoreExpectedEOF(nil, nil); err != nil {
		t.Fatal(err)
	}
}

func TestIgnoreExpectedEOFWithEOFErrorButNoCloseHint(t *testing.T) {
	resp := &http.Response{}
	in := composeWithEOFError("antani")
	if err := ignoreExpectedEOF(in, resp); !errors.Is(err, io.EOF) {
		t.Fatalf("not the error we expected: %+v", err)
	}
}

func TestIgnoreExpectedEOFWithEOFErrorAndCloseHint(t *testing.T) {
	resp := &http.Response{Close: true}
	in := composeWithEOFError("antani")
	if err := ignoreExpectedEOF(in, resp); err != nil {
		t.Fatal(err)
	}
}

func TestIgnoreExpectedEOFAnyOtherErrorAndCloseHint(t *testing.T) {
	resp := &http.Response{Close: true}
	in := errors.New("antani")
	if err := ignoreExpectedEOF(in, resp); !errors.Is(err, in) {
		t.Fatalf("not the error we expected: %+v", err)
	}
}

func TestIgnoreExpectedEOFAnyOtherErrorAndNoCloseHint(t *testing.T) {
	resp := &http.Response{Close: false /*explicit*/}
	in := errors.New("antani")
	if err := ignoreExpectedEOF(in, resp); !errors.Is(err, in) {
		t.Fatalf("not the error we expected: %+v", err)
	}
}
