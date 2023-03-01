package netem

// Bandwidth is the maximum data transfer across a given path.
type Bandwidth int64

// BitsPerSecond is the constant to multiply [Bandwidth] for so that
// the measurement unit is bits per second.
const BitsPerSecond = 1

// KilobitsPerSecond is like [BitsPerSecond] but for kbit/s.
const KilobitsPerSecond = 1000

// MegabitsPerSecond is like [KilobitsPerSecond] but for Mbit/s.
const MegabitsPerSecond = 1_000_000
