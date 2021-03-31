package assetsx

import (
	"net"
	"strings"
	"testing"

	"github.com/oschwald/geoip2-golang"
)

func TestASN(t *testing.T) {
	data := Must(ASNDatabaseData())
	db, err := geoip2.FromBytes(data)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	record, err := db.ASN(net.ParseIP("8.8.8.8"))
	if err != nil {
		t.Fatal(err)
	}
	t.Log(record)
}

func TestCountry(t *testing.T) {
	data := Must(CountryDatabaseData())
	db, err := geoip2.FromBytes(data)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	record, err := db.Country(net.ParseIP("8.8.8.8"))
	if err != nil {
		t.Fatal(err)
	}
	t.Log(record)
}

func TestGzipReaderError(t *testing.T) {
	m := &manager{}
	data, err := m.read([]byte("foobarbaz"))
	if err == nil || !strings.HasSuffix(err.Error(), "unexpected EOF") {
		t.Fatal("not the error we expected", err)
	}
	if data != nil {
		t.Fatal("expected nil data")
	}
}
