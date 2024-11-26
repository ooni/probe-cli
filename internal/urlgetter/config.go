package urlgetter

// Config contains the configuration.
type Config struct {
	// HTTPHost allows overriding the default HTTP host.
	HTTPHost string `ooni:"Force using specific HTTP Host header"`

	// HTTPReferer sets the HTTP referer value.
	HTTPReferer string `ooni:"Force using the specific HTTP Referer header"`

	// Method selects the HTTP method to use.
	Method string `ooni:"Force HTTP method different than GET"`

	// NoFollowRedirects disables following redirects.
	NoFollowRedirects bool `ooni:"Disable following redirects"`

	// TLSNextProtos is an OPTIONAL comma separated ALPN list.
	TLSNextProtos string `ooni:"Comma-separated list of next protocols for ALPN"`

	// TLSServerName is the OPTIONAL SNI value.
	TLSServerName string `ooni:"SNI value to use"`
}

// Clone returns a deep copy of the given [*Config].
func (cx *Config) Clone() *Config {
	return &Config{
		HTTPHost:          cx.HTTPHost,
		HTTPReferer:       cx.HTTPReferer,
		Method:            cx.Method,
		NoFollowRedirects: cx.NoFollowRedirects,
		TLSNextProtos:     cx.TLSNextProtos,
		TLSServerName:     cx.TLSServerName,
	}
}
