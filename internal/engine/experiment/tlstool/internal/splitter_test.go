package internal_test

import (
	"testing"

	"github.com/ooni/probe-cli/v3/internal/engine/experiment/tlstool/internal"
	"github.com/ooni/probe-cli/v3/internal/randx"
)

func TestSplitter84restSmall(t *testing.T) {
	input := []byte("1111222")
	output := internal.Splitter84rest(input)
	if len(output) != 1 {
		t.Fatal("invalid output length")
	}
	if string(output[0]) != "1111222" {
		t.Fatal("invalid output[0]")
	}
}

func TestSplitter84restGood(t *testing.T) {
	input := []byte("1111222233334")
	output := internal.Splitter84rest(input)
	if len(output) != 3 {
		t.Fatal("invalid output length")
	}
	if string(output[0]) != "11112222" {
		t.Fatal("invalid output[0]")
	}
	if string(output[1]) != "3333" {
		t.Fatal("invalid output[1]")
	}
	if string(output[2]) != "4" {
		t.Fatal("invalid output[2]")
	}
}

func TestSplitter3264randSmall(t *testing.T) {
	input := randx.Letters(64)
	output := internal.Splitter3264rand([]byte(input))
	if len(output) != 1 {
		t.Fatal("invalid output length")
	}
	if string(output[0]) != input {
		t.Fatal("invalid output[0]")
	}
}

func TestSplitter3264Works(t *testing.T) {
	input := randx.Letters(65)
	output := internal.Splitter3264rand([]byte(input))
	for i := 0; i < 32; i++ {
		if len(output) != 2 {
			t.Fatal("invalid output length")
		}
		if len(output[0]) < 32 || len(output[0]) > 64 {
			t.Fatal("invalid output[0] length")
		}
	}
}

func TestSNISplitterEasyCase(t *testing.T) {
	input := []byte("11112222334555foo.barbar.deadbeef.com6777778888")
	sni := []byte("barbar.deadbeef.com")
	output := internal.SNISplitter(input, sni)
	if len(output) != 9 {
		t.Fatal("invalid output length")
	}
	if string(output[0]) != "11112222334555foo." {
		t.Fatal("invalid output[0]")
	}
	if string(output[1]) != "bar" {
		t.Fatal("invalid output[1]")
	}
	if string(output[2]) != "bar" {
		t.Fatal("invalid output[2]")
	}
	if string(output[3]) != ".de" {
		t.Fatal("invalid output[3]")
	}
	if string(output[4]) != "adb" {
		t.Fatal("invalid output[4]")
	}
	if string(output[5]) != "eef" {
		t.Fatal("invalid output[5]")
	}
	if string(output[6]) != ".co" {
		t.Fatal("invalid output[6]")
	}
	if string(output[7]) != "m" {
		t.Fatal("invalid output[7]")
	}
	if string(output[8]) != "6777778888" {
		t.Fatal("invalid output[8]")
	}
}

func TestSNISplitterNoMatch(t *testing.T) {
	input := []byte("11112222334555foo.barbar.deadbeef.com6777778888")
	sni := []byte("www.google.com")
	output := internal.SNISplitter(input, sni)
	if len(output) != 1 {
		t.Fatal("invalid output length")
	}
	if string(output[0]) != string(input) {
		t.Fatal("invalid output[0]")
	}
}

func TestSNISplitterWithUnicode(t *testing.T) {
	input := []byte("11112222334555你好世界.com6777778888")
	sni := []byte("你好世界.com")
	output := internal.SNISplitter(input, sni)
	t.Log(string(output[2]))
	t.Log(output)
	if len(output) != 8 {
		t.Fatal("invalid output length")
	}
	if string(output[0]) != "11112222334555" {
		t.Fatal("invalid output[0]")
	}
	if string(output[1]) != "你" {
		t.Fatal("invalid output[1]")
	}
	if string(output[2]) != "好" {
		t.Fatal("invalid output[2]")
	}
	if string(output[3]) != "世" {
		t.Fatal("invalid output[3]")
	}
	if string(output[4]) != "界" {
		t.Fatal("invalid output[4]")
	}
	if string(output[5]) != ".co" {
		t.Fatal("invalid output[5]")
	}
	if string(output[6]) != "m" {
		t.Fatal("invalid output[6]")
	}
	if string(output[7]) != "6777778888" {
		t.Fatal("invalid output[7]")
	}
}
