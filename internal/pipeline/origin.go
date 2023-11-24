package pipeline

// Origin indicates the header origin
type Origin int64

const (
	// OriginProbe indicates that the header was seen by the probe
	OriginProbe = Origin(1 << iota)

	// OriginTH indicates that the header was seen by the TH
	OriginTH
)
