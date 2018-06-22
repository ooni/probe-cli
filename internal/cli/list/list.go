package list

import (
	"fmt"

	"github.com/alecthomas/kingpin"
	"github.com/apex/log"
	"github.com/ooni/probe-cli/internal/cli/root"
	"github.com/ooni/probe-cli/internal/database"
	"github.com/ooni/probe-cli/internal/output"
)

func init() {
	cmd := root.Command("list", "List results")

	resultID := cmd.Arg("id", "the id of the result to list measurements for").Int64()

	cmd.Action(func(_ *kingpin.ParseContext) error {
		ctx, err := root.Init()
		if err != nil {
			log.WithError(err).Error("failed to initialize root context")
			return err
		}
		if *resultID > 0 {
			measurements, err := database.ListMeasurements(ctx.DB, *resultID)
			if err != nil {
				log.WithError(err).Error("failed to list measurements")
				return err
			}
			for idx, msmt := range measurements {
				fmt.Printf("%d: %v\n", idx, msmt)
			}
		} else {
			doneResults, incompleteResults, err := database.ListResults(ctx.DB)
			if err != nil {
				log.WithError(err).Error("failed to list results")
				return err
			}

			if len(incompleteResults) > 0 {
				output.SectionTitle("Incomplete results")
			}
			for idx, result := range incompleteResults {
				output.ResultItem(output.ResultItemData{
					ID:            result.ID,
					Index:         idx,
					TotalCount:    len(incompleteResults),
					Name:          result.Name,
					StartTime:     result.StartTime,
					NetworkName:   result.NetworkName,
					Country:       result.Country,
					ASN:           result.ASN,
					Summary:       result.Summary,
					Done:          result.Done,
					DataUsageUp:   result.DataUsageUp,
					DataUsageDown: result.DataUsageDown,
				})
			}

			resultSummary := output.ResultSummaryData{}
			netCount := make(map[string]int)
			output.SectionTitle("Results")
			for idx, result := range doneResults {
				output.ResultItem(output.ResultItemData{
					ID:            result.ID,
					Index:         idx,
					TotalCount:    len(doneResults),
					Name:          result.Name,
					StartTime:     result.StartTime,
					NetworkName:   result.NetworkName,
					Country:       result.Country,
					ASN:           result.ASN,
					Summary:       result.Summary,
					Done:          result.Done,
					DataUsageUp:   result.DataUsageUp,
					DataUsageDown: result.DataUsageDown,
				})
				resultSummary.TotalTests++
				netCount[result.ASN]++
				resultSummary.TotalDataUsageUp += result.DataUsageUp
				resultSummary.TotalDataUsageDown += result.DataUsageDown
			}
			resultSummary.TotalNetworks = int64(len(netCount))

			output.ResultSummary(resultSummary)
		}

		return nil
	})
}
