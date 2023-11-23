package pipeline

// HeaderOrigin indicates the header origin
type HeaderOrigin int64

const (
	// HeaderOriginProbe indicates that the header was seen by the probe
	HeaderOriginProbe = HeaderOrigin(1 << iota)

	// HeaderOriginTH indicates that the header was seen by the TH
	HeaderOriginTH
)
