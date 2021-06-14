package ooapi

import (
	"strings"
	"testing"
)

func TestDefaultTemplateExecutorParseError(t *testing.T) {
	te := &defaultTemplateExecutor{}
	out, err := te.Execute("{{ .Foo", nil)
	if err == nil || !strings.HasSuffix(err.Error(), "unclosed action") {
		t.Fatal("not the error we expected", err)
	}
	if out != "" {
		t.Fatal("expected empty string")
	}
}

func TestDefaultTemplateExecutorExecError(t *testing.T) {
	te := &defaultTemplateExecutor{}
	arg := make(chan interface{})
	out, err := te.Execute("{{ .Foo }}", arg)
	if err == nil || !strings.Contains(err.Error(), `can't evaluate field Foo`) {
		t.Fatal("not the error we expected", err)
	}
	if out != "" {
		t.Fatal("expected empty string")
	}
}

func TestDefaultGobCodecEncodeError(t *testing.T) {
	codec := &defaultGobCodec{}
	arg := make(chan interface{})
	data, err := codec.Encode(arg)
	if err == nil || !strings.Contains(err.Error(), "can't handle type") {
		t.Fatal("not the error we expected", err)
	}
	if data != nil {
		t.Fatal("expected nil data")
	}
}
