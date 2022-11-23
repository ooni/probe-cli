package rm

import (
	"errors"
	"fmt"

	"github.com/AlecAivazis/survey/v2"
	"github.com/alecthomas/kingpin"
	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/cmd/ooniprobe/internal/cli/root"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/upper/db/v4"
)

func deleteAll(d model.WritableDatabase, r model.ReadableDatabase, skipInteractive bool) error {
	if skipInteractive == false {
		answer := ""
		confirm := &survey.Select{
			Message: fmt.Sprintf("Are you sure you wish to delete ALL results"),
			Options: []string{"true", "false"},
			Default: "false",
		}
		survey.AskOne(confirm, &answer, nil)
		if answer == "false" {
			return errors.New("canceled by user")
		}
	}
	doneResults, incompleteResults, err := r.ListResults()
	if err != nil {
		log.WithError(err).Error("failed to list results")
		return err
	}
	cnt := 0
	for _, result := range incompleteResults {
		err = d.DeleteResult(result.DatabaseResult.ID)
		if err == db.ErrNoMoreRows {
			log.WithError(err).Errorf("failed to delete result #%d", result.DatabaseResult.ID)
		}
		cnt++
	}
	for _, result := range doneResults {
		err = d.DeleteResult(result.DatabaseResult.ID)
		if err == db.ErrNoMoreRows {
			log.WithError(err).Errorf("failed to delete result #%d", result.DatabaseResult.ID)
		}
		cnt++
	}
	log.Infof("Deleted #%d measurements", cnt)
	return nil
}

func init() {
	cmd := root.Command("rm", "Delete a result")
	yes := cmd.Flag("yes", "Skip interactive prompt").Bool()
	all := cmd.Flag("all", "Delete all measurements").Bool()

	resultID := cmd.Arg("id", "the id of the result to delete").Int64()

	cmd.Action(func(_ *kingpin.ParseContext) error {
		ctx, err := root.Init()
		if err != nil {
			log.Errorf("%s", err)
			return err
		}

		if *all == true {
			return deleteAll(ctx.WriteDB(), ctx.ReadDB(), *yes)
		}

		if *yes == true {
			err = ctx.WriteDB().DeleteResult(*resultID)
			if err == db.ErrNoMoreRows {
				return errors.New("result not found")
			}
			return err
		}
		answer := ""
		confirm := &survey.Select{
			Message: fmt.Sprintf("Are you sure you wish to delete the result #%d", *resultID),
			Options: []string{"true", "false"},
			Default: "false",
		}
		survey.AskOne(confirm, &answer, nil)
		if answer == "false" {
			return errors.New("canceled by user")
		}
		err = ctx.WriteDB().DeleteResult(*resultID)
		if err == db.ErrNoMoreRows {
			return errors.New("result not found")
		}
		return err
	})
}
