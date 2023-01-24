package dslx

import (
	"errors"
	"net"
	"testing"
)

func TestAddressSet(t *testing.T) {
	initAddr := "93.184.216.34"
	dns := []*Maybe[*ResolvedAddresses]{
		{
			Error: nil,
			State: &ResolvedAddresses{
				Addresses: []string{initAddr},
				Domain:    "example.com",
			},
		},
		{
			Error: errors.New("mocked"),
			State: nil,
		},
	}
	// NewAddressSet
	as := NewAddressSet(dns...)
	if len(as.M) != 1 {
		t.Fatalf("NewAdressSet: invalid number of entries")
	}
	if !as.M[initAddr] {
		t.Fatalf("NewAdressSet: invalid entry")
	}
	// Add
	bogonIP := "a.b.c.d"
	as.Add(bogonIP)
	if len(as.M) != 2 {
		t.Fatalf("Add: invalid number of entries")
	}
	if !as.M[bogonIP] {
		t.Fatalf("Add: invalid entry")
	}
	// RemoveBogons
	as.RemoveBogons()
	if len(as.M) != 1 {
		t.Fatalf("RemoveBogons: invalid number of entries")
	}
	if as.M[bogonIP] {
		t.Fatalf("RemoveBogons: invalid entry")
	}
	// ToEndpoints
	endpoints := as.ToEndpoints("tcp", 443)
	if len(endpoints) != 1 {
		t.Fatalf("ToEndpoints: invalid number of entries")
	}
	if endpoints[0].Address != net.JoinHostPort(initAddr, "443") {
		t.Fatalf("ToEndpoints: invalid endpoint address %s", endpoints[0].Address)
	}
}
