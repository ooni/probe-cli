package measurex

//
// Origin
//
// Here we define the origin type.
//

// Origin is the origin of a measurement.
type Origin string

var (
	// OriginProbe means that the probe performed this measurement.
	OriginProbe = Origin("probe")

	// OriginTH means that the test helper performed this measurement.
	OriginTH = Origin("th")
)
