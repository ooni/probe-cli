package output

import (
	"bufio"
	"fmt"
	"os"
	"time"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/internal/util"
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
	ID                      int64
	Name                    string
	StartTime               time.Time
	TestKeys                string
	MeasurementCount        uint64
	MeasurementAnomalyCount uint64
	Runtime                 float64
	Country                 string
	NetworkName             string
	ASN                     string
	Done                    bool
	DataUsageDown           int64
	DataUsageUp             int64
	Index                   int
	TotalCount              int
}

// ResultItem logs a progress type event
func ResultItem(result ResultItemData) {
	log.WithFields(log.Fields{
		"type":                      "result_item",
		"id":                        result.ID,
		"name":                      result.Name,
		"start_time":                result.StartTime,
		"test_keys":                 result.TestKeys,
		"measurement_count":         result.MeasurementCount,
		"measurement_anomaly_count": result.MeasurementAnomalyCount,
		"country":                   result.Country,
		"network_name":              result.NetworkName,
		"asn":                       result.ASN,
		"runtime":                   result.Runtime,
		"done":                      result.Done,
		"data_usage_down":           result.DataUsageDown,
		"data_usage_up":             result.DataUsageUp,
		"index":                     result.Index,
		"total_count":               result.TotalCount,
	}).Info("result item")
}

type ResultSummaryData struct {
	TotalTests         int64
	TotalDataUsageUp   int64
	TotalDataUsageDown int64
	TotalNetworks      int64
}

func ResultSummary(result ResultSummaryData) {
	log.WithFields(log.Fields{
		"type":                  "result_summary",
		"total_tests":           result.TotalTests,
		"total_data_usage_up":   result.TotalDataUsageUp,
		"total_data_usage_down": result.TotalDataUsageDown,
		"total_networks":        result.TotalNetworks,
	}).Info("result summary")
}

// SectionTitle is the title of a section
func SectionTitle(text string) {
	log.WithFields(log.Fields{
		"type":  "section_title",
		"title": text,
	}).Info(text)
}

func Paragraph(text string) {
	const width = 80
	fmt.Println(util.WrapString(text, width))
}

func Bullet(text string) {
	const width = 80
	fmt.Printf("• %s\n", util.WrapString(text, width))
}

func PressEnterToContinue(text string) {
	fmt.Print(text)
	bufio.NewReader(os.Stdin).ReadBytes('\n')
}
