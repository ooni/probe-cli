package rm

import (
	"errors"
	"fmt"

	"github.com/alecthomas/kingpin"
	"github.com/apex/log"
	"github.com/ooni/probe-cli/internal/cli/root"
	"github.com/ooni/probe-cli/internal/database"
	survey "gopkg.in/AlecAivazis/survey.v1"
	db "upper.io/db.v3"
)

func init() {
	cmd := root.Command("rm", "Delete a result")
	yes := cmd.Flag("yes", "Skip interactive prompt").Bool()

	resultID := cmd.Arg("id", "the id of the result to delete").Int64()

	cmd.Action(func(_ *kingpin.ParseContext) error {
		ctx, err := root.Init()
		if err != nil {
			log.Errorf("%s", err)
			return err
		}

		if *yes == true {
			err = database.DeleteResult(ctx.DB, *resultID)
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
		err = database.DeleteResult(ctx.DB, *resultID)
		if err == db.ErrNoMoreRows {
			return errors.New("result not found")
		}
		return err
	})
}
