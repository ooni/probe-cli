package summary

import "fmt"

// ResultSummaryFunc is the function used to generate result summaries
type ResultSummaryFunc func(SummaryMap) (string, error)

// SummaryMap contains a mapping from test name to serialized summary for it
type SummaryMap map[string][]string

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

func CheckRequiredKeys(rk []string, m SummaryMap) error {
	for _, key := range rk {
		if _, ok := m[key]; ok {
			continue
		}
		return fmt.Errorf("missing SummaryMap key '%s'", key)
	}
	return nil
}
