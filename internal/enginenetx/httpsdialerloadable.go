package enginenetx

import (
	"encoding/json"
)

// HTTPSDialerLoadablePolicy is an [HTTPSDialerPolicy] that you
// can load from its JSON serialization on disk.
type HTTPSDialerLoadablePolicy struct {
	// Domains maps each domain to its policy. When there is
	// no domain, the code falls back to the default "null" policy
	// implemented by the HTTPSDialerNullPolicy struct.
	Domains map[string][]*HTTPSDialerTactic
}

// LoadHTTPSDialerPolicy loads the [HTTPSDialerPolicy] from
// the given bytes containing a serialized JSON object.
func LoadHTTPSDialerPolicy(data []byte) (*HTTPSDialerLoadablePolicy, error) {
	var p HTTPSDialerLoadablePolicy
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, err
	}
	return &p, nil
}
