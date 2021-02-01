package crashreport

import (
	"github.com/apex/log"
	"github.com/getsentry/raven-go"
)

// Disabled flag is used to globally disable crash reporting and make all the
// crash reporting logic a no-op.
var Disabled = false

var client *raven.Client

// CapturePanic is a wrapper around raven.CapturePanic that becomes a noop if
// `Disabled` is set to true.
func CapturePanic(f func(), tags map[string]string) (interface{}, string) {
	if Disabled == true {
		return nil, ""
	}
	return client.CapturePanic(f, tags)
}

// CapturePanicAndWait is a wrapper around raven.CapturePanicAndWait that becomes a noop if
// `Disabled` is set to true.
func CapturePanicAndWait(f func(), tags map[string]string) (interface{}, string) {
	if Disabled == true {
		return nil, ""
	}
	return client.CapturePanicAndWait(f, tags)
}

// CaptureError is a wrapper around raven.CaptureError
func CaptureError(err error, tags map[string]string) string {
	if Disabled == true {
		return ""
	}
	return client.CaptureError(err, tags)
}

// CaptureErrorAndWait is a wrapper around raven.CaptureErrorAndWait
func CaptureErrorAndWait(err error, tags map[string]string) string {
	if Disabled == true {
		return ""
	}
	return client.CaptureErrorAndWait(err, tags)
}

// Wait will block on sending messages to the sentry server
func Wait() {
	if Disabled == false {
		log.Info("sending exception backtrace")
		client.Wait()
	}
}

func init() {
	var err error
	client, err = raven.NewClient("https://cb4510e090f64382ac371040c19b2258:8448daeebfa643c289ef398f8645980b@sentry.io/1234954", nil)
	if err != nil {
		log.WithError(err).Error("failed to create a raven client")
	}
}
