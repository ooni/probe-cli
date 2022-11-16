package list

import (
	"strings"

	"github.com/alecthomas/kingpin"
	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/cmd/ooniprobe/internal/cli/root"
	"github.com/ooni/probe-cli/v3/cmd/ooniprobe/internal/output"
)

func init() {
	cmd := root.Command("list", "List results")
	resultID := cmd.Arg("id", "the id of the result to list measurements for").Int64()
	cmd.Action(func(_ *kingpin.ParseContext) error {
		probeCLI, err := root.Init()
		if err != nil {
			log.WithError(err).Error("failed to initialize root context")
			return err
		}
		if *resultID > 0 {
			measurements, err := probeCLI.DB().ListMeasurements(*resultID)
			if err != nil {
				log.WithError(err).Error("failed to list measurements")
				return err
			}
			msmtSummary := output.MeasurementSummaryData{
				TotalCount:         0,
				AnomalyCount:       0,
				DataUsageUp:        0.0,
				DataUsageDown:      0.0,
				TotalRuntime:       0,
				ASN:                0,
				NetworkName:        "",
				NetworkCountryCode: "ZZ",
			}
			isFirst := true
			isLast := false
			for idx, msmt := range measurements {
				if idx > 0 {
					isFirst = false
				}
				if idx == len(measurements)-1 {
					isLast = true
				}
				// We assume that since these are summary level information the first
				// item will contain the information necessary.
				if isFirst {
					msmtSummary.TotalRuntime = msmt.Result.Runtime
					msmtSummary.DataUsageUp = msmt.DataUsageUp
					msmtSummary.DataUsageDown = msmt.DataUsageDown
					msmtSummary.NetworkName = msmt.NetworkName
					msmtSummary.NetworkCountryCode = msmt.Network.CountryCode
					msmtSummary.ASN = msmt.ASN
					msmtSummary.StartTime = msmt.Measurement.StartTime
				}
				if msmt.IsAnomaly.Bool == true {
					msmtSummary.AnomalyCount++
				}
				msmtSummary.TotalCount++
				output.MeasurementItem(msmt, isFirst, isLast)
			}
			output.MeasurementSummary(msmtSummary)
		} else {
			doneResults, incompleteResults, err := probeCLI.DB().ListResults()
			if err != nil {
				log.WithError(err).Error("failed to list results")
				return err
			}
			if len(incompleteResults) > 0 {
				output.SectionTitle("Incomplete results")
			}
			for idx, result := range incompleteResults {
				output.ResultItem(output.ResultItemData{
					ID:                      result.Result.ID,
					Index:                   idx,
					TotalCount:              len(incompleteResults),
					Name:                    result.TestGroupName,
					StartTime:               result.StartTime,
					NetworkName:             result.Network.NetworkName,
					Country:                 result.Network.CountryCode,
					ASN:                     result.Network.ASN,
					MeasurementCount:        0,
					MeasurementAnomalyCount: 0,
					TestKeys:                "{}", // FIXME this used to be Summary we probably need to use a list now
					Done:                    result.IsDone,
					IsUploaded:              result.IsUploaded,
					DataUsageUp:             result.DataUsageUp,
					DataUsageDown:           result.DataUsageDown,
				})
			}
			resultSummary := output.ResultSummaryData{}
			netCount := make(map[uint]int)
			output.SectionTitle("Results")
			for idx, result := range doneResults {
				testKeys := "{}"

				// We only care to expose in the testKeys the value of the ndt test result
				if result.TestGroupName == "performance" {
					// The test_keys column are concanetated with the "|" character as a separator.
					// We consider this to be safe since we only really care about values of the
					// performance test_keys where the values are all numbers and none of the keys
					// contain the "|" character.
					for _, e := range strings.Split(result.TestKeys, "|") {
						// We use the presence of the "download" key to indicate we have found the
						// ndt test_keys, since the dash result does not contain it.
						if strings.Contains(e, "download") {
							testKeys = e
						}
					}
				}

				output.ResultItem(output.ResultItemData{
					ID:                      result.Result.ID,
					Index:                   idx,
					TotalCount:              len(doneResults),
					Name:                    result.TestGroupName,
					StartTime:               result.StartTime,
					NetworkName:             result.Network.NetworkName,
					Country:                 result.Network.CountryCode,
					ASN:                     result.Network.ASN,
					TestKeys:                testKeys,
					MeasurementCount:        result.TotalCount,
					MeasurementAnomalyCount: result.AnomalyCount,
					Done:                    result.IsDone,
					DataUsageUp:             result.DataUsageUp,
					DataUsageDown:           result.DataUsageDown,
				})
				resultSummary.TotalTests++
				netCount[result.Network.ASN]++
				resultSummary.TotalDataUsageUp += result.DataUsageUp
				resultSummary.TotalDataUsageDown += result.DataUsageDown
			}
			resultSummary.TotalNetworks = int64(len(netCount))
			output.ResultSummary(resultSummary)
		}
		return nil
	})
}
