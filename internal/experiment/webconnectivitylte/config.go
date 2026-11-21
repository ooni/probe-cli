package webconnectivitylte

//
// Config
//

// Config contains webconnectivity experiment configuration.
type Config struct {
	DNSOverUDPResolver string `json:"dns_over_udp_resolver,omitempty" ooni:"DNS-over-UDP resolver endpoint to use (e.g. '8.8.8.8:53')"`
}
