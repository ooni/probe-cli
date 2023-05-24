package mocks

import "testing"

func TestLocationProvider(t *testing.T) {
	t.Run("ProbeASN", func(t *testing.T) {
		expected := uint(1)
		loc := LocationProvider{
			MockProbeASN: func() uint {
				return expected
			},
		}
		r := loc.ProbeASN()
		if r != expected {
			t.Fatal("not the uint we expected")
		}
	})

	t.Run("ProbeASNString", func(t *testing.T) {
		expected := "mocked"
		loc := LocationProvider{
			MockProbeASNString: func() string {
				return expected
			},
		}
		r := loc.ProbeASNString()
		if r != expected {
			t.Fatal("not the string we expected")
		}
	})

	t.Run("ProbeCC", func(t *testing.T) {
		expected := "mocked"
		loc := LocationProvider{
			MockProbeCC: func() string {
				return expected
			},
		}
		r := loc.ProbeCC()
		if r != expected {
			t.Fatal("not the string we expected")
		}
	})

	t.Run("ProbeIP", func(t *testing.T) {
		expected := "mocked"
		loc := LocationProvider{
			MockProbeIP: func() string {
				return expected
			},
		}
		r := loc.ProbeIP()
		if r != expected {
			t.Fatal("not the string we expected")
		}
	})

	t.Run("ProbeNetworkName", func(t *testing.T) {
		expected := "mocked"
		loc := LocationProvider{
			MockProbeNetworkName: func() string {
				return expected
			},
		}
		r := loc.ProbeNetworkName()
		if r != expected {
			t.Fatal("not the string we expected")
		}
	})
	t.Run("ResolverIP", func(t *testing.T) {
		expected := "mocked"
		loc := LocationProvider{
			MockResolverIP: func() string {
				return expected
			},
		}
		r := loc.ResolverIP()
		if r != expected {
			t.Fatal("not the string we expected")
		}
	})
}
