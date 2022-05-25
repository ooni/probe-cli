package netxlite

import (
	"bytes"
	"encoding/binary"
	"strings"
	"testing"

	"github.com/miekg/dns"
	"github.com/ooni/probe-cli/v3/internal/randx"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

func TestDNSEncoderMiekg(t *testing.T) {
	t.Run("we can fail to encode a domain name to bytes", func(t *testing.T) {
		e := &DNSEncoderMiekg{}
		domain := randx.LettersUppercase(512)
		query := e.Encode(domain, dns.TypeA, false)
		data, err := query.Bytes()
		if err == nil || !strings.HasSuffix(err.Error(), "bad rdata") {
			t.Fatal("unexpected err", err)
		}
		if data != nil {
			t.Fatal("expected nil data here")
		}
	})

	t.Run("calls to bytes are memoized", func(t *testing.T) {
		t.Run("on success", func(t *testing.T) {
			e := &DNSEncoderMiekg{}
			query := e.Encode("x.org", dns.TypeA, false)
			checkResult := func(data []byte, err error) {
				if err != nil {
					t.Fatal("unexpected err", err)
				}
				dnsValidateEncodedQueryBytes(t, data, byte(dns.TypeA), query.ID())
			}
			const repeat = 3
			for idx := 0; idx < repeat; idx++ {
				checkResult(query.Bytes())
			}
			// The following cast will always work in this configuration
			if query.(*dnsQuery).bytesCalls.Load() != 1 {
				t.Fatal("invalid number of calls")
			}
		})

		t.Run("on failure", func(t *testing.T) {
			e := &DNSEncoderMiekg{}
			domain := randx.LettersUppercase(512)
			query := e.Encode(domain, dns.TypeA, false)
			checkResult := func(data []byte, err error) {
				if err == nil || !strings.HasSuffix(err.Error(), "bad rdata") {
					t.Fatal("unexpected err", err)
				}
				if data != nil {
					t.Fatal("expected nil data here")
				}
			}
			const repeat = 3
			for idx := 0; idx < repeat; idx++ {
				checkResult(query.Bytes())
			}
			// The following cast will always work in this configuration
			if query.(*dnsQuery).bytesCalls.Load() != repeat {
				t.Fatal("invalid number of calls")
			}
		})
	})

	t.Run("encode A", func(t *testing.T) {
		e := &DNSEncoderMiekg{}
		query := e.Encode("x.org", dns.TypeA, false)
		if query.Domain() != "x.org" {
			t.Fatal("invalid domain")
		}
		if query.Type() != dns.TypeA {
			t.Fatal("invalid type")
		}
		data, err := query.Bytes()
		if err != nil {
			t.Fatal(err)
		}
		dnsValidateEncodedQueryBytes(t, data, byte(dns.TypeA), query.ID())
	})

	t.Run("encode AAAA", func(t *testing.T) {
		e := &DNSEncoderMiekg{}
		query := e.Encode("x.org", dns.TypeAAAA, false)
		if query.Domain() != "x.org" {
			t.Fatal("invalid domain")
		}
		if query.Type() != dns.TypeAAAA {
			t.Fatal("invalid type")
		}
		data, err := query.Bytes()
		if err != nil {
			t.Fatal(err)
		}
		dnsValidateEncodedQueryBytes(t, data, byte(dns.TypeA), query.ID())
	})

	t.Run("encode padding", func(t *testing.T) {
		// The purpose of this unit test is to make sure that for a wide
		// array of values we obtain the right query size.
		getquerylen := func(domainlen int, padding bool) int {
			e := &DNSEncoderMiekg{}
			query := e.Encode(
				// This is not a valid name because it ends up being way
				// longer than 255 octets. However, the library is allowing
				// us to generate such name and we are not going to send
				// it on the wire. Also, we check below that the query that
				// we generate is long enough, so we should be good.
				dns.Fqdn(strings.Repeat("x.", domainlen)),
				dns.TypeA, padding,
			)
			data, err := query.Bytes()
			if err != nil {
				t.Fatal(err)
			}
			return len(data)
		}
		for domainlen := 1; domainlen <= 4000; domainlen++ {
			vanillalen := getquerylen(domainlen, false)
			paddedlen := getquerylen(domainlen, true)
			if vanillalen < domainlen {
				t.Fatal("vanillalen is smaller than domainlen")
			}
			if (paddedlen % dnsPaddingDesiredBlockSize) != 0 {
				t.Fatal("paddedlen is not a multiple of PaddingDesiredBlockSize")
			}
			if paddedlen < vanillalen {
				t.Fatal("paddedlen is smaller than vanillalen")
			}
		}
	})
}

// dnsValidateEncodedQueryBytes validates the query serialized in data
// for the given query type qtype (e.g., dns.TypeAAAA).
func dnsValidateEncodedQueryBytes(t *testing.T, data []byte, qtype byte, qid uint16) {
	var wirequery uint16
	err := binary.Read(bytes.NewReader(data), binary.BigEndian, &wirequery)
	runtimex.PanicOnError(err, "binary.Read failed unexpectedly")
	if wirequery != qid {
		t.Fatal("invalid query ID")
	}
	if data[2] != 1 {
		t.Fatal("FLAGS should only have RD set")
	}
	if data[3] != 0 {
		t.Fatal("RA|Z|Rcode should be zero")
	}
	if data[4] != 0 || data[5] != 1 {
		t.Fatal("QCOUNT high should be one")
	}
	if data[6] != 0 || data[7] != 0 {
		t.Fatal("ANCOUNT should be zero")
	}
	if data[8] != 0 || data[9] != 0 {
		t.Fatal("NSCOUNT should be zero")
	}
	if data[10] != 0 || data[11] != 0 {
		t.Fatal("ARCOUNT should be zero")
	}
	t.Log(data[12])
	if data[12] != 1 || data[13] != byte('x') {
		t.Fatal("The name does not contain 1:x")
	}
	if data[14] != 3 || data[15] != byte('o') || data[16] != byte('r') || data[17] != byte('g') {
		t.Fatal("The name does not contain 3:org")
	}
	if data[18] != 0 {
		t.Fatal("The name does not terminate where expected")
	}
	if data[19] != 0 && data[20] != qtype {
		t.Fatal("The query is not for the expected type")
	}
	if data[21] != 0 && data[22] != 1 {
		t.Fatal("The query is not IN")
	}
}
