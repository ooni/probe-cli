package im

import (
	"github.com/measurement-kit/go-measurement-kit"
	"github.com/ooni/probe-cli/nettests"
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
	Blocked      bool
}

// Summary generates a summary for a test run
func (h Telegram) Summary(tk map[string]interface{}) interface{} {
	var (
		tcpBlocking  bool
		httpBlocking bool
		webBlocking  bool
	)

	if tk["telegram_tcp_blocking"] == nil {
		tcpBlocking = false
	} else {
		tcpBlocking = tk["telegram_tcp_blocking"].(bool)
	}
	if tk["telegram_http_blocking"] == nil {
		httpBlocking = false
	} else {
		httpBlocking = tk["telegram_http_blocking"].(bool)
	}
	if tk["telegram_web_status"] == nil {
		webBlocking = false
	} else {
		webBlocking = tk["telegram_web_status"].(string) == "blocked"
	}

	return TelegramSummary{
		TCPBlocking:  tcpBlocking,
		HTTPBlocking: httpBlocking,
		WebBlocking:  webBlocking,
		Blocked:      webBlocking || httpBlocking || tcpBlocking,
	}
}

// LogSummary writes the summary to the standard output
func (h Telegram) LogSummary(s string) error {
	return nil
}
