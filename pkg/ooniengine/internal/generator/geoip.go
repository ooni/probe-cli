package main

//
// GeoIP task
//

func init() {
	basename := "GeoIP"

	registerNewEvent(
		"Probe geolocation information.",
		basename,
		StructField{
			Docs: []string{
				"Failure that occurred or empty string (on success)",
			},
			Name: "Failure",
			Type: TypeString,
		},
		StructField{
			Docs: []string{
				"The probe's IP address.",
			},
			Name: "ProbeIP",
			Type: TypeString,
		},
		StructField{
			Docs: []string{
				"ASN derived from the probe's IP.",
			},
			Name: "ProbeASN",
			Type: TypeString,
		},
		StructField{
			Docs: []string{
				"Country code derived from the probe's IP.",
			},
			Name: "ProbeCC",
			Type: TypeString,
		},
		StructField{
			Docs: []string{
				"Network name of the probe's ASN.",
			},
			Name: "ProbeNetworkName",
			Type: TypeString,
		},
		StructField{
			Docs: []string{
				"IPv4 address used by getaddrinfo.",
			},
			Name: "ResolverIP",
			Type: TypeString,
		},
		StructField{
			Docs: []string{
				"ASN derived from the resolver's IP.",
			},
			Name: "ResolverASN",
			Type: TypeString,
		},
		StructField{
			Docs: []string{
				"Network name of resolver's ASN.",
			},
			Name: "ResolverNetworkName",
			Type: TypeString,
		},
	)

	geoIPConfig := registerNewConfig(
		"Contains config for the GeoIP task.",
		basename,
		StructField{
			Docs: []string{
				"Config for creating a session.",
			},
			Name: "Session",
			Type: "SessionConfig",
		},
	)

	OONIEngine.Tasks = append(OONIEngine.Tasks, Task{
		Name:   basename,
		Config: geoIPConfig,
	})
}
