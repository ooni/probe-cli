package groups

import (
	"encoding/json"

	"github.com/apex/log"
	"github.com/openobservatory/gooni/internal/database"
	"github.com/openobservatory/gooni/nettests"
	"github.com/openobservatory/gooni/nettests/performance"
	"github.com/openobservatory/gooni/nettests/websites"
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

// NettestGroups that can be run by the user
var NettestGroups = map[string]NettestGroup{
	"websites": NettestGroup{
		Label: "Websites",
		Nettests: []nettests.Nettest{
			websites.WebConnectivity{},
		},
		Summary: func(m database.SummaryMap) (string, error) {
			return "{}", nil
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
			err = json.Unmarshal([]byte(m["Dash"]), &dashSummary)
			if err != nil {
				log.WithError(err).Error("failed to unmarshal Dash summary")
				return "", err
			}
			err = json.Unmarshal([]byte(m["Ndt"]), &ndtSummary)
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
	"middleboxes": NettestGroup{
		Label:    "Middleboxes",
		Nettests: []nettests.Nettest{},
		Summary: func(m database.SummaryMap) (string, error) {
			return "{}", nil
		},
	},
	"im": NettestGroup{
		Label:    "Instant Messaging",
		Nettests: []nettests.Nettest{},
		Summary: func(m database.SummaryMap) (string, error) {
			return "{}", nil
		},
	},
}
