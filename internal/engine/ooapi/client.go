package ooapi

// Client is a client for speaking with the OONI API. Make sure you
// fill in the mandatory fields when you create a Client.
type Client struct {
	BaseURL      string       // optional
	GobCodec     GobCodec     // optional
	HTTPClient   HTTPClient   // optional
	JSONCodec    JSONCodec    // optional
	KVStore      KVStore      // mandatory
	RequestMaker RequestMaker // optional
	UserAgent    string       // optional
}
