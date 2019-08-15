package im

import (
	"github.com/ooni/probe-cli/nettests"
	"github.com/ooni/probe-engine/experiment/whatsapp"
)

// WhatsApp test implementation
type WhatsApp struct {
}

// Run starts the test
func (h WhatsApp) Run(ctl *nettests.Controller) error {
	experiment := whatsapp.NewExperiment(ctl.Ctx.Session, whatsapp.Config{
		LogLevel: "INFO",
	})
	return ctl.Run(experiment, []string{""})
}

// WhatsAppTestKeys for the test
type WhatsAppTestKeys struct {
	RegistrationServerBlocking bool `json:"registration_server_blocking"`
	WebBlocking                bool `json:"whatsapp_web_blocking"`
	EndpointsBlocking          bool `json:"whatsapp_endpoints_blocking"`
	IsAnomaly                  bool `json:"-"`
}

// GetTestKeys generates a summary for a test run
func (h WhatsApp) GetTestKeys(tk map[string]interface{}) (interface{}, error) {
	var (
		webBlocking          bool
		registrationBlocking bool
		endpointsBlocking    bool
	)

	var computeBlocking = func(key string) bool {
		const blk = "blocked"
		if tk[key] == nil {
			return false
		}
		if tk[key].(string) == blk {
			return true
		}
		return false
	}
	registrationBlocking = computeBlocking("registration_server_status")
	webBlocking = computeBlocking("whatsapp_web_status")
	endpointsBlocking = computeBlocking("whatsapp_endpoints_status")

	return WhatsAppTestKeys{
		RegistrationServerBlocking: registrationBlocking,
		WebBlocking:                webBlocking,
		EndpointsBlocking:          endpointsBlocking,
		IsAnomaly:                  registrationBlocking || webBlocking || endpointsBlocking,
	}, nil
}

// LogSummary writes the summary to the standard output
func (h WhatsApp) LogSummary(s string) error {
	return nil
}
