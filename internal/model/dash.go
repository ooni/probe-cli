package model

import "fmt"

//
// neubot/dash experiment data model.
//

// TODO(bassosimone): before merging modify DASH to use
// this definition instead of its own definition.

// DASHNegotiateResponse contains the response of DASH negotiation.
type DASHNegotiateResponse struct {
	Authorization string `json:"authorization"`
	QueuePos      int64  `json:"queue_pos"`
	RealAddress   string `json:"real_address"`
	Unchoked      int    `json:"unchoked"`
}

// DASHMinSize is the minimum segment size that this server can return.
//
// The client requests two second chunks. The minimum emulated streaming
// speed is the minimum streaming speed (in kbit/s) multiplied by 1000
// to obtain bit/s, divided by 8 to obtain bytes/s and multiplied by the
// two seconds to obtain the minimum segment size.
const DASHMinSize = 100 * 1000 / 8 * 2

// DASHMaxSize is the maximum segment size that this server can return. See
// the docs of MinSize for more information on how it is computed.
const DASHMaxSize = 30000 * 1000 / 8 * 2

// DASHMinSizeString is [dashMinSize] as a string.
var DASHMinSizeString = fmt.Sprintf("%d", DASHMinSize)
