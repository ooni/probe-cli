package ndt7

import "time"

const (
	paramFractionForScaling   = 16
	paramMinMessageSize       = 1 << 10
	paramMaxBufferSize        = 1 << 20
	paramMaxScaledMessageSize = 1 << 20
	paramMaxMessageSize       = 1 << 24
	paramMaxRuntimeUpperBound = 15.0 // seconds
	paramMaxRuntime           = 10 * time.Second
	paramMeasureInterval      = 250 * time.Millisecond
)
