package output

import (
	"time"

	"github.com/apex/log"
)

// Progress logs a progress type event
func Progress(key string, perc float64, msg string) {
	log.WithFields(log.Fields{
		"type":       "progress",
		"key":        key,
		"percentage": perc,
	}).Info(msg)
}

// ResultItemData is the metadata about a result
type ResultItemData struct {
	ID            int64
	Name          string
	StartTime     time.Time
	Summary       string
	Runtime       float64
	Country       string
	NetworkName   string
	ASN           string
	Done          bool
	DataUsageDown int64
	DataUsageUp   int64
	Index         int
	TotalCount    int
}

// ResultItem logs a progress type event
func ResultItem(result ResultItemData) {
	log.WithFields(log.Fields{
		"type":            "result_item",
		"id":              result.ID,
		"name":            result.Name,
		"start_time":      result.StartTime,
		"summary":         result.Summary,
		"country":         result.Country,
		"network_name":    result.NetworkName,
		"asn":             result.ASN,
		"runtime":         result.Runtime,
		"done":            result.Done,
		"data_usage_down": result.DataUsageDown,
		"data_usage_up":   result.DataUsageUp,
		"index":           result.Index,
		"total_count":     result.TotalCount,
	}).Info("result item")
}
