package main

//
// Metrics definitions
//

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// metricRequestsTotal counts the total number of requests
	metricRequestsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "oohelperd_requests_total",
		Help: "The total number of processed requests",
	})

	// metricRequestsByStatusCode counts the number of requests that
	// have returned a given status code to the caller.
	metricRequestsByStatusCode = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "oohelperd_requests_by_status_code",
		Help: "Total number of processed requests by status code",
	}, []string{"code", "reason"})

	// metricRequestsInflight counts the number of requests currently inflight.
	metricRequestsInflight = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "oohelperd_requests_inflight",
		Help: "The number or requests currently inflight",
	})

	// metricMeasurementTime summarizes the time to perform a measurement.
	metricMeasurementTime = promauto.NewSummary(prometheus.SummaryOpts{
		Name: "oohelperd_measurement_time",
		Help: "Summarizes the time to perform a test-helper measurement (in seconds)",
		// See https://grafana.com/blog/2022/03/01/how-summary-metrics-work-in-prometheus/
		Objectives: map[float64]float64{
			0.25: 0.010, // 0.240 <= φ <= 0.260
			0.5:  0.010, // 0.490 <= φ <= 0.510
			0.75: 0.010, // 0.740 <= φ <= 0.760
			0.9:  0.010, // 0.899 <= φ <= 0.901
			0.99: 0.001, // 0.989 <= φ <= 0.991
		},
	})

	// metricMeasurementCount counts the number of calls to measure.
	metricMeasurementCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "oohelperd_measurement_count",
		Help: "The total number of test-helper measurements performed",
	})

	// metricMeasurementFailed counts the number of times that measure failed.
	metricMeasurementFailed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "oohelperd_measurement_failed",
		Help: "The number of test-helper measurements that failed",
	})
)
