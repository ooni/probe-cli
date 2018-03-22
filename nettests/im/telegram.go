package im

import (
	"github.com/measurement-kit/go-measurement-kit"
	"github.com/openobservatory/gooni/nettests"
)

// Telegram test implementation
type Telegram struct {
}

// Run starts the test
func (h Telegram) Run(ctl *nettests.Controller) error {
	mknt := mk.NewNettest("Telegram")
	ctl.Init(mknt)
	return mknt.Run()
}

// TelegramSummary for the test
type TelegramSummary struct {
	HTTPBlocking bool
	TCPBlocking  bool
	WebBlocking  bool
}

// Summary generates a summary for a test run
func (h Telegram) Summary(tk map[string]interface{}) interface{} {
	return TelegramSummary{
		TCPBlocking:  tk["telegram_tcp_blocking"].(bool) == true,
		HTTPBlocking: tk["telegram_http_blocking"].(bool) == true,
		WebBlocking:  tk["telegram_web_status"].(string) == "blocked",
	}
}

// LogSummary writes the summary to the standard output
func (h Telegram) LogSummary(s string) error {
	return nil
}
