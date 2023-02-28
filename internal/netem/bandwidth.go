package netem

//
// Definition of the `Bandwidth`` type and of constants useful to
// express the speed of point-to-point links.
//

// Bandwidth is the bandwidth in bit/s.
type Bandwidth int64

// BitPerSecond is the constant to scale [Bandwidth] to obtain bit/s.
const BitPerSecond = 1

// KiloBitPerSecond is the constant to scale [Bandwidth] to obtain kbit/s.
const KiloBitPerSecond = 1000

// MegaBitPerSecond is the constant to scale [Bandwidth] to Mbit/s.
const MegaBitPerSecond = 100 * KiloBitPerSecond

// GigaBitPerSecond is the constant to scale [Bandwidth] to Gbit/s.
const GigaBitPerSecond = 100 * MegaBitPerSecond
