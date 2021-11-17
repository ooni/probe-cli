package dash

const (
	// negotiatePath is the URL path used to negotiate
	negotiatePath = "/negotiate/dash"

	// downloadPath is the URL path used to request DASH segments. You can
	// append to this path an integer indicating how many bytes you would like
	// the server to send you as part of the next chunk.
	downloadPath = "/dash/download/"

	// collectPath is the URL path used to collect
	collectPath = "/collect/dash"
)

// defaultRates contains the default DASH rates in kbit/s.
var defaultRates = []int64{
	100, 150, 200, 250, 300, 400, 500, 700, 900, 1200, 1500, 2000,
	2500, 3000, 4000, 5000, 6000, 7000, 10000, 20000,
}
