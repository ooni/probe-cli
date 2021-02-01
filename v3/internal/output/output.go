package output

import (
	"bufio"
	"fmt"
	"os"
	"time"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/database"
	"github.com/ooni/probe-cli/v3/internal/utils"
)

// MeasurementJSON prints the JSON of a measurement
func MeasurementJSON(j map[string]interface{}) {
	log.WithFields(log.Fields{
		"type":             "measurement_json",
		"measurement_json": j,
	}).Info("Measurement JSON")
}

// Progress logs a progress type event
func Progress(key string, perc float64, eta float64, msg string) {
	log.WithFields(log.Fields{
		"type":       "progress",
		"key":        key,
		"percentage": perc,
		"eta":        eta,
	}).Info(msg)
}

type MeasurementSummaryData struct {
	TotalRuntime       float64
	TotalCount         int64
	AnomalyCount       int64
	DataUsageUp        float64
	DataUsageDown      float64
	ASN                uint
	NetworkName        string
	NetworkCountryCode string
	StartTime          time.Time
}

func MeasurementSummary(msmt MeasurementSummaryData) {
	log.WithFields(log.Fields{
		"type":                 "measurement_summary",
		"total_runtime":        msmt.TotalRuntime,
		"total_count":          msmt.TotalCount,
		"anomaly_count":        msmt.AnomalyCount,
		"data_usage_down":      msmt.DataUsageDown,
		"data_usage_up":        msmt.DataUsageUp,
		"asn":                  msmt.ASN,
		"network_country_code": msmt.NetworkCountryCode,
		"network_name":         msmt.NetworkName,
		"start_time":           msmt.StartTime,
	}).Info("measurement summary")
}

// MeasurementItem logs a progress type event
func MeasurementItem(msmt database.MeasurementURLNetwork, isFirst bool, isLast bool) {
	log.WithFields(log.Fields{
		"type":     "measurement_item",
		"is_first": isFirst,
		"is_last":  isLast,

		"id":                    msmt.Measurement.ID,
		"test_name":             msmt.TestName,
		"test_group_name":       msmt.Result.TestGroupName,
		"start_time":            msmt.Measurement.StartTime,
		"test_keys":             msmt.TestKeys,
		"network_country_code":  msmt.Network.CountryCode,
		"network_name":          msmt.Network.NetworkName,
		"asn":                   msmt.Network.ASN,
		"runtime":               msmt.Measurement.Runtime,
		"url":                   msmt.URL.URL.String,
		"url_category_code":     msmt.URL.CategoryCode.String,
		"url_country_code":      msmt.URL.CountryCode.String,
		"is_anomaly":            msmt.IsAnomaly.Bool,
		"is_uploaded":           msmt.IsUploaded,
		"is_upload_failed":      msmt.IsUploadFailed,
		"upload_failure_msg":    msmt.UploadFailureMsg.String,
		"is_failed":             msmt.IsFailed,
		"failure_msg":           msmt.FailureMsg.String,
		"is_done":               msmt.Measurement.IsDone,
		"report_file_path":      msmt.ReportFilePath.String,
		"measurement_file_path": msmt.MeasurementFilePath.String,
	}).Info("measurement")
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
	ASN                     uint
	Done                    bool
	DataUsageDown           float64
	DataUsageUp             float64
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
		"network_country_code":      result.Country,
		"network_name":              result.NetworkName,
		"asn":                       result.ASN,
		"runtime":                   result.Runtime,
		"is_done":                   result.Done,
		"data_usage_down":           result.DataUsageDown,
		"data_usage_up":             result.DataUsageUp,
		"index":                     result.Index,
		"total_count":               result.TotalCount,
	}).Info("result item")
}

type ResultSummaryData struct {
	TotalTests         int64
	TotalDataUsageUp   float64
	TotalDataUsageDown float64
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
	fmt.Println(utils.WrapString(text, width))
}

func Bullet(text string) {
	const width = 80
	fmt.Printf("• %s\n", utils.WrapString(text, width))
}

func PressEnterToContinue(text string) error {
	fmt.Print(text)
	_, err := bufio.NewReader(os.Stdin).ReadBytes('\n')
	return err
}
