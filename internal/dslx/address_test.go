package dslx

import (
	"errors"
	"net"
	"testing"
)

/*
Test cases:
- Create new address set:
  - with valid input
  - with invalid input

- Add address:
  - on empty set
  - on non-empty set

- Remove bogons:
  - on empty set
  - on bogon set
  - on valid set

- Convert address set to endpoints list
*/
func TestAddressSet(t *testing.T) {
	t.Run("Create new address set", func(t *testing.T) {
		t.Run("with valid input", func(t *testing.T) {
			initAddr := "93.184.216.34"
			dns := []*Maybe[*ResolvedAddresses]{
				{
					Error: nil,
					State: &ResolvedAddresses{
						Addresses: []string{initAddr},
						Domain:    "example.com",
					},
				},
			}
			as := NewAddressSet(dns...)
			if len(as.M) != 1 {
				t.Fatalf("invalid number of entries")
			}
			if !as.M[initAddr] {
				t.Fatalf("invalid entry")
			}
		})

		t.Run("with invalid input", func(t *testing.T) {
			dns := []*Maybe[*ResolvedAddresses]{
				{
					Error: errors.New("mocked"),
					State: nil,
				},
			}
			as := NewAddressSet(dns...)
			if len(as.M) != 0 {
				t.Fatalf("invalid number of entries")
			}
		})
	})

	type addressTest struct {
		as   *AddressSet
		want int
	}

	t.Run("Add address", func(t *testing.T) {
		var tests = map[string]addressTest{
			"on empty set":     {as: &AddressSet{map[string]bool{}}, want: 1},
			"on non-empty set": {as: &AddressSet{map[string]bool{"1.1.1.1": true}}, want: 2},
		}
		for name, tt := range tests {
			t.Run(name, func(t *testing.T) {
				ip := "5.4.3.2"
				tt.as.Add(ip)
				if len(tt.as.M) != tt.want {
					t.Fatalf("invalid number of entries")
				}
				if !tt.as.M[ip] {
					t.Fatalf("invalid entry")
				}
			})
		}
	})

	t.Run("Remove bogons", func(t *testing.T) {
		bogonIP := "10.0.0.1"
		var tests = map[string]addressTest{
			"on empty set": {as: &AddressSet{map[string]bool{}}, want: 0},
			"on bogon set": {as: &AddressSet{map[string]bool{bogonIP: true}}, want: 0},
			"on valid set": {as: &AddressSet{map[string]bool{"1.1.1.1": true}}, want: 1},
		}
		for name, tt := range tests {
			t.Run(name, func(t *testing.T) {
				tt.as.RemoveBogons()
				if len(tt.as.M) != tt.want {
					t.Fatalf("invalid number of entries")
				}
				if tt.as.M[bogonIP] {
					t.Fatalf("bogon entry still exists")
				}
			})
		}
	})

	t.Run("Convert address set to endpoints list", func(t *testing.T) {
		as := &AddressSet{map[string]bool{"1.1.1.1": true}}
		endpoints := as.ToEndpoints("tcp", 443)
		if len(endpoints) != 1 {
			t.Fatalf("invalid number of entries")
		}
		_, port, _ := net.SplitHostPort(endpoints[0].Address)
		if port != "443" {
			t.Fatalf("invalid port in endpoint address %s", endpoints[0].Address)
		}
	})
}
