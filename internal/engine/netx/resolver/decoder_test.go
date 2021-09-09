package resolver

import (
	"strings"
	"testing"

	"github.com/miekg/dns"
)

func TestDecoderUnpackError(t *testing.T) {
	d := &MiekgDecoder{}
	data, err := d.Decode(dns.TypeA, nil)
	if err == nil {
		t.Fatal("expected an error here")
	}
	if data != nil {
		t.Fatal("expected nil data here")
	}
}

func TestDecoderNXDOMAIN(t *testing.T) {
	d := &MiekgDecoder{}
	data, err := d.Decode(dns.TypeA, GenReplyError(t, dns.RcodeNameError))
	if err == nil || !strings.HasSuffix(err.Error(), "no such host") {
		t.Fatal("not the error we expected")
	}
	if data != nil {
		t.Fatal("expected nil data here")
	}
}

func TestDecoderOtherError(t *testing.T) {
	d := &MiekgDecoder{}
	data, err := d.Decode(dns.TypeA, GenReplyError(t, dns.RcodeRefused))
	if err == nil || !strings.HasSuffix(err.Error(), "query failed") {
		t.Fatal("not the error we expected")
	}
	if data != nil {
		t.Fatal("expected nil data here")
	}
}

func TestDecoderNoAddress(t *testing.T) {
	d := &MiekgDecoder{}
	data, err := d.Decode(dns.TypeA, GenReplySuccess(t, dns.TypeA))
	if err == nil || !strings.HasSuffix(err.Error(), "no response returned") {
		t.Fatal("not the error we expected")
	}
	if data != nil {
		t.Fatal("expected nil data here")
	}
}

func TestDecoderDecodeA(t *testing.T) {
	d := &MiekgDecoder{}
	data, err := d.Decode(
		dns.TypeA, GenReplySuccess(t, dns.TypeA, "1.1.1.1", "8.8.8.8"))
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != 2 {
		t.Fatal("expected two entries here")
	}
	if data[0] != "1.1.1.1" {
		t.Fatal("invalid first IPv4 entry")
	}
	if data[1] != "8.8.8.8" {
		t.Fatal("invalid second IPv4 entry")
	}
}

func TestDecoderDecodeAAAA(t *testing.T) {
	d := &MiekgDecoder{}
	data, err := d.Decode(
		dns.TypeAAAA, GenReplySuccess(t, dns.TypeAAAA, "::1", "fe80::1"))
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != 2 {
		t.Fatal("expected two entries here")
	}
	if data[0] != "::1" {
		t.Fatal("invalid first IPv6 entry")
	}
	if data[1] != "fe80::1" {
		t.Fatal("invalid second IPv6 entry")
	}
}

func TestDecoderUnexpectedAReply(t *testing.T) {
	d := &MiekgDecoder{}
	data, err := d.Decode(
		dns.TypeA, GenReplySuccess(t, dns.TypeAAAA, "::1", "fe80::1"))
	if err == nil || !strings.HasSuffix(err.Error(), "no response returned") {
		t.Fatal("not the error we expected")
	}
	if data != nil {
		t.Fatal("expected nil data here")
	}
}

func TestDecoderUnexpectedAAAAReply(t *testing.T) {
	d := &MiekgDecoder{}
	data, err := d.Decode(
		dns.TypeAAAA, GenReplySuccess(t, dns.TypeA, "1.1.1.1", "8.8.4.4."))
	if err == nil || !strings.HasSuffix(err.Error(), "no response returned") {
		t.Fatal("not the error we expected")
	}
	if data != nil {
		t.Fatal("expected nil data here")
	}
}
