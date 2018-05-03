package list

import (
	"github.com/alecthomas/kingpin"
	"github.com/apex/log"
	"github.com/ooni/probe-cli/internal/cli/root"
	"github.com/ooni/probe-cli/internal/database"
	"github.com/ooni/probe-cli/internal/output"
)

func init() {
	cmd := root.Command("list", "List results")

	cmd.Action(func(_ *kingpin.ParseContext) error {
		ctx, err := root.Init()
		if err != nil {
			log.WithError(err).Error("failed to initialize root context")
			return err
		}
		doneResults, incompleteResults, err := database.ListResults(ctx.DB)
		if err != nil {
			log.WithError(err).Error("failed to list results")
			return err
		}

		log.Info("Results")
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
		}
		log.Info("Incomplete results")
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
		return nil
	})
}
