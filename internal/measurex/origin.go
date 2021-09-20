package measurex

// Origin is the origin of a measurement.
type Origin string

// These are the possible origins.
var (
	OriginProbe = Origin("probe")
	OriginTH    = Origin("th")
)
