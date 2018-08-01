package crashreport

import (
	"github.com/getsentry/raven-go"
)

// Disabled flag is used to globally disable crash reporting and make all the
// crash reporting logic a no-op.
var Disabled = false

// CapturePanic is a wrapper around raven.CapturePanic that becomes a noop if
// `Disabled` is set to true.
func CapturePanic(f func(), tags map[string]string) (interface{}, string) {
	if Disabled == true {
		return nil, ""
	}
	return raven.CapturePanic(f, tags)
}

// CapturePanicAndWait is a wrapper around raven.CapturePanicAndWait that becomes a noop if
// `Disabled` is set to true.
func CapturePanicAndWait(f func(), tags map[string]string) (interface{}, string) {
	if Disabled == true {
		return nil, ""
	}
	return raven.CapturePanicAndWait(f, tags)
}

// CaptureError is a wrapper around raven.CaptureError
func CaptureError(err error, tags map[string]string) string {
	if Disabled == true {
		return ""
	}
	return raven.CaptureError(err, tags)
}

// CaptureErrorAndWait is a wrapper around raven.CaptureErrorAndWait
func CaptureErrorAndWait(err error, tags map[string]string) string {
	if Disabled == true {
		return ""
	}
	return raven.CaptureErrorAndWait(err, tags)
}

func init() {
	raven.SetDSN("https://cb4510e090f64382ac371040c19b2258:8448daeebfa643c289ef398f8645980b@sentry.io/1234954")
}
