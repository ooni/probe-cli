package ooapi

// Client is a client for speaking with the OONI API. Make sure you
// fill in the mandatory fields.
type Client struct {
	// KVStore is the MANDATORY key-value store. You can use
	// the kvstore.Memory{} struct for an in-memory store.
	KVStore KVStore

	// The following fields are optional. When they are empty
	// we will fallback to sensible defaults.
	BaseURL      string
	GobCodec     GobCodec
	HTTPClient   HTTPClient
	JSONCodec    JSONCodec
	RequestMaker RequestMaker
	UserAgent    string
}
