// Package internal contains the implementation of tlstool.
package internal

import (
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine/netx"
)

// DialerConfig contains the config for creating a dialer
type DialerConfig struct {
	Dialer netx.Dialer
	Delay  time.Duration
	SNI    string
}

// NewSNISplitterDialer creates a new dialer that splits
// outgoing messages such that the SNI should end up being
// splitted into different TCP segments.
func NewSNISplitterDialer(config DialerConfig) Dialer {
	return Dialer{
		Dialer: config.Dialer,
		Delay:  config.Delay,
		Splitter: func(b []byte) [][]byte {
			return SNISplitter(b, []byte(config.SNI))
		},
	}
}

// NewThriceSplitterDialer creates a new dialer that splits
// outgoing messages in three parts according to the circumvention
// technique described by Kevin Boch in the Internet Measurement
// Village 2020 <https://youtu.be/ksojSRFLbBM?t=1140>.
func NewThriceSplitterDialer(config DialerConfig) Dialer {
	return Dialer{
		Dialer:   config.Dialer,
		Delay:    config.Delay,
		Splitter: Splitter84rest,
	}
}

// NewRandomSplitterDialer creates a new dialer that splits
// the SNI like the fixed splitting schema used by outline. See
// github.com/Jigsaw-Code/outline-go-tun2socks.
func NewRandomSplitterDialer(config DialerConfig) Dialer {
	return Dialer{
		Dialer:   config.Dialer,
		Delay:    config.Delay,
		Splitter: Splitter3264rand,
	}
}

// NewVanillaDialer creates a new vanilla dialer that does
// nothing and is used to establish a baseline.
func NewVanillaDialer(config DialerConfig) Dialer {
	return Dialer{Dialer: config.Dialer}
}
