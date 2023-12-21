package oohelperd

//
// Metrics definitions
//
// See https://github.com/ooni/probe/issues/2183#issuecomment-1230327725
//

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// metricsSummaryObjectives returns the summary objectives for promauto.NewSummary.
func metricsSummaryObjectives() map[float64]float64 {
	// See https://grafana.com/blog/2022/03/01/how-summary-metrics-work-in-prometheus/
	//
	// TODO(bassosimone,FedericoCeratto): investigate whether using
	// a shorter-than-10m observation interval is better for us
	return map[float64]float64{
		0.25: 0.010, // 0.240 <= φ <= 0.260
		0.5:  0.010, // 0.490 <= φ <= 0.510
		0.75: 0.010, // 0.740 <= φ <= 0.760
		0.9:  0.010, // 0.899 <= φ <= 0.901
		0.99: 0.001, // 0.989 <= φ <= 0.991
	}
}

var (
	// metricRequestsCount counts the number of requests we served.
	metricRequestsCount = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "oohelperd_requests_count",
		Help: "Total number of processed requests",
	}, []string{"code", "reason"})

	// metricRequestsInflight gauges the number of requests currently inflight.
	metricRequestsInflight = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "oohelperd_requests_inflight_gauge",
		Help: "The number or requests currently inflight",
	})

	// metricWCTaskDurationSeconds summarizes the duration of the web connectivity measurement task.
	metricWCTaskDurationSeconds = promauto.NewSummary(prometheus.SummaryOpts{
		Name:       "oohelperd_wctask_duration_seconds",
		Help:       "Summarizes the time to complete the Web Connectivity measurement task (in seconds)",
		Objectives: metricsSummaryObjectives(),
	})

	// metricDNSTaskDurationSeconds summarizes the duration of the DNS task.
	metricDNSTaskDurationSeconds = promauto.NewSummary(prometheus.SummaryOpts{
		Name:       "oohelperd_dnstask_duration_seconds",
		Help:       "Summarizes the time to complete the DNS measurement task (in seconds)",
		Objectives: metricsSummaryObjectives(),
	})

	// metricTCPTaskDurationSeconds summarizes the duration of the TCP task.
	metricTCPTaskDurationSeconds = promauto.NewSummary(prometheus.SummaryOpts{
		Name:       "oohelperd_tcptask_duration_seconds",
		Help:       "Summarizes the time to complete the TCP measurement task (in seconds)",
		Objectives: metricsSummaryObjectives(),
	})

	// metricTLSTaskDurationSeconds summarizes the duration of the TLS task.
	metricTLSTaskDurationSeconds = promauto.NewSummary(prometheus.SummaryOpts{
		Name:       "oohelperd_tlstask_duration_seconds",
		Help:       "Summarizes the time to complete the TLS measurement task (in seconds)",
		Objectives: metricsSummaryObjectives(),
	})

	// metricHTTPTaskDurationSeconds summarizes the duration of the HTTP task.
	metricHTTPTaskDurationSeconds = promauto.NewSummary(prometheus.SummaryOpts{
		Name:       "oohelperd_httptask_duration_seconds",
		Help:       "Summarizes the time to complete the HTTP measurement task (in seconds)",
		Objectives: metricsSummaryObjectives(),
	})
)
