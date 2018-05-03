package groups

import (
	"encoding/json"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/internal/database"
	"github.com/ooni/probe-cli/nettests"
	"github.com/ooni/probe-cli/nettests/im"
	"github.com/ooni/probe-cli/nettests/middlebox"
	"github.com/ooni/probe-cli/nettests/performance"
	"github.com/ooni/probe-cli/nettests/websites"
)

// NettestGroup base structure
type NettestGroup struct {
	Label    string
	Nettests []nettests.Nettest
	Summary  database.ResultSummaryFunc
}

// PerformanceSummary is the result summary for a performance test
type PerformanceSummary struct {
	Upload   int64
	Download int64
	Ping     float64
	Bitrate  int64
}

// MiddleboxSummary is the summary for the middlebox tests
type MiddleboxSummary struct {
	Detected bool
}

// IMSummary is the summary for the im tests
type IMSummary struct {
	Tested  uint
	Blocked uint
}

// WebsitesSummary is the summary for the websites test
type WebsitesSummary struct {
	Tested  uint
	Blocked uint
}

// NettestGroups that can be run by the user
var NettestGroups = map[string]NettestGroup{
	"websites": NettestGroup{
		Label: "Websites",
		Nettests: []nettests.Nettest{
			websites.WebConnectivity{},
		},
		Summary: func(m database.SummaryMap) (string, error) {
			// XXX to generate this I need to create the summary map as a list
			var summary WebsitesSummary
			summary.Tested = 0
			summary.Blocked = 0
			for _, msmtSummaryStr := range m["WebConnectivity"] {
				var wcSummary websites.WebConnectivitySummary

				err := json.Unmarshal([]byte(msmtSummaryStr), &wcSummary)
				if err != nil {
					log.WithError(err).Error("failed to unmarshal WebConnectivity summary")
					return "", err
				}
				if wcSummary.Blocked {
					summary.Blocked++
				}
				summary.Tested++
			}
			summaryBytes, err := json.Marshal(summary)
			if err != nil {
				return "", err
			}
			return string(summaryBytes), nil
		},
	},
	"performance": NettestGroup{
		Label: "Performance",
		Nettests: []nettests.Nettest{
			performance.Dash{},
			performance.NDT{},
		},
		Summary: func(m database.SummaryMap) (string, error) {
			var (
				err         error
				ndtSummary  performance.NDTSummary
				dashSummary performance.DashSummary
				summary     PerformanceSummary
			)
			err = json.Unmarshal([]byte(m["Dash"][0]), &dashSummary)
			if err != nil {
				log.WithError(err).Error("failed to unmarshal Dash summary")
				return "", err
			}
			err = json.Unmarshal([]byte(m["Ndt"][0]), &ndtSummary)
			if err != nil {
				log.WithError(err).Error("failed to unmarshal NDT summary")
				return "", err
			}
			summary.Bitrate = dashSummary.Bitrate
			summary.Download = ndtSummary.Download
			summary.Upload = ndtSummary.Upload
			summary.Ping = ndtSummary.AvgRTT
			summaryBytes, err := json.Marshal(summary)
			if err != nil {
				return "", err
			}
			return string(summaryBytes), nil
		},
	},
	"middlebox": NettestGroup{
		Label: "Middleboxes",
		Nettests: []nettests.Nettest{
			middlebox.HTTPInvalidRequestLine{},
			middlebox.HTTPHeaderFieldManipulation{},
		},
		Summary: func(m database.SummaryMap) (string, error) {
			var (
				err         error
				hhfmSummary middlebox.HTTPHeaderFieldManipulationSummary
				hirlSummary middlebox.HTTPInvalidRequestLineSummary
				summary     MiddleboxSummary
			)
			err = json.Unmarshal([]byte(m["HttpHeaderFieldManipulation"][0]), &hhfmSummary)
			if err != nil {
				log.WithError(err).Error("failed to unmarshal hhfm summary")
				return "", err
			}
			err = json.Unmarshal([]byte(m["HttpInvalidRequestLine"][0]), &hirlSummary)
			if err != nil {
				log.WithError(err).Error("failed to unmarshal hirl summary")
				return "", err
			}
			summary.Detected = hirlSummary.Tampering == true || hhfmSummary.Tampering == true
			summaryBytes, err := json.Marshal(summary)
			if err != nil {
				return "", err
			}
			return string(summaryBytes), nil
		},
	},
	"im": NettestGroup{
		Label: "Instant Messaging",
		Nettests: []nettests.Nettest{
			im.FacebookMessenger{},
			im.Telegram{},
			im.WhatsApp{},
		},
		Summary: func(m database.SummaryMap) (string, error) {
			var (
				err       error
				waSummary im.WhatsAppSummary
				tgSummary im.TelegramSummary
				fbSummary im.FacebookMessengerSummary
				summary   IMSummary
			)
			err = json.Unmarshal([]byte(m["Whatsapp"][0]), &waSummary)
			if err != nil {
				log.WithError(err).Error("failed to unmarshal whatsapp summary")
				return "", err
			}
			err = json.Unmarshal([]byte(m["Telegram"][0]), &tgSummary)
			if err != nil {
				log.WithError(err).Error("failed to unmarshal telegram summary")
				return "", err
			}
			err = json.Unmarshal([]byte(m["FacebookMessenger"][0]), &fbSummary)
			if err != nil {
				log.WithError(err).Error("failed to unmarshal facebook summary")
				return "", err
			}
			// XXX it could actually be that some are not tested when the
			// configuration is changed.
			summary.Tested = 3
			summary.Blocked = 0
			if fbSummary.Blocked == true {
				summary.Blocked++
			}
			if tgSummary.Blocked == true {
				summary.Blocked++
			}
			if waSummary.Blocked == true {
				summary.Blocked++
			}

			summaryBytes, err := json.Marshal(summary)
			if err != nil {
				return "", err
			}
			return string(summaryBytes), nil
		},
	},
}
