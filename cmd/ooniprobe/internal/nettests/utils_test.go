package nettests

import (
	"strings"
	"testing"
)

func TestStringListToModelURLInfoWithValidInput(t *testing.T) {
	input := []string{
		"stun://stun.voip.blackberry.com:3478",
		"stun://stun.altar.com.pl:3478",
	}
	output, err := stringListToModelURLInfo(input)
	if err != nil {
		t.Fatal(err)
	}
	if len(input) != len(output) {
		t.Fatal("unexpected output length")
	}
	for idx := 0; idx < len(input); idx++ {
		if input[idx] != output[idx].URL {
			t.Fatal("unexpected entry")
		}
		if output[idx].CategoryCode != "MISC" {
			t.Fatal("unexpected category")
		}
		if output[idx].CountryCode != "XX" {
			t.Fatal("unexpected country")
		}
	}
}

func TestStringListToModelURLInfoWithInvalidInput(t *testing.T) {
	input := []string{
		"stun://stun.voip.blackberry.com:3478",
		"\t",
		"stun://stun.altar.com.pl:3478",
	}
	output, err := stringListToModelURLInfo(input)
	if err == nil || !strings.HasSuffix(err.Error(), "invalid control character in URL") {
		t.Fatal("no the error we expected", err)
	}
	if output != nil {
		t.Fatal("unexpected nil output")
	}
}
