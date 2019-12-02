package nettests

// Telegram test implementation
type Telegram struct {
}

// Run starts the test
func (h Telegram) Run(ctl *Controller) error {
	builder, err := ctl.Ctx.Session.NewExperimentBuilder(
		"telegram",
	)
	if err != nil {
		return err
	}
	if err := builder.SetOptionString("LogLevel", "INFO"); err != nil {
		return err
	}
	return ctl.Run(builder, []string{""})
}

// TelegramTestKeys for the test
type TelegramTestKeys struct {
	HTTPBlocking bool `json:"telegram_http_blocking"`
	TCPBlocking  bool `json:"telegram_tcp_blocking"`
	WebBlocking  bool `json:"telegram_web_blocking"`
	IsAnomaly    bool `json:"-"`
}

// GetTestKeys generates a summary for a test run
func (h Telegram) GetTestKeys(tk map[string]interface{}) (interface{}, error) {
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

	return TelegramTestKeys{
		TCPBlocking:  tcpBlocking,
		HTTPBlocking: httpBlocking,
		WebBlocking:  webBlocking,
		IsAnomaly:    webBlocking || httpBlocking || tcpBlocking,
	}, nil
}

// LogSummary writes the summary to the standard output
func (h Telegram) LogSummary(s string) error {
	return nil
}
