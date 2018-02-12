package legacy

import (
	"fmt"
	"os"
	"path/filepath"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"gopkg.in/AlecAivazis/survey.v1"
)

func HomePath() (string, error) {
	home, err := homedir.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".ooni"), nil
}

// HomeExists returns true if a legacy home exists
func HomeExists() (bool, error) {
	home, err := HomePath()
	if err != nil {
		return false, err
	}
	path := filepath.Join(home, "ooniprobe.conf")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false, nil
	}
	return true, nil
}

// BackupHome the legacy home directory
func BackupHome() error {
	home, err := homedir.Dir()
	if err != nil {
		return errors.Wrap(err, "backing up home")
	}
	oldPath := filepath.Join(home, ".ooni")
	newPath := filepath.Join(home, ".ooni-legacy")
	if err := os.Rename(oldPath, newPath); err != nil {
		return errors.Wrap(err, "backing up home")
	}
	return nil
}

// MaybeMigrateHome prompts the user if we should backup the legacy home
func MaybeMigrateHome() error {
	exists, err := HomeExists()
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	home, err := HomePath()
	if err != nil {
		return err
	}
	logf("We found an existing OONI Probe installation")
	chosen := ""
	prompt := &survey.Select{
		Message: "Should we:",
		Options: []string{"delete it", "back it up"},
	}
	survey.AskOne(prompt, &chosen, nil)
	if chosen == "delete it" {
		if err := os.RemoveAll(home); err != nil {
			return err
		}
	} else {
		logf("Backing up ~/.ooni to ~/.ooni-legacy")
		if err := BackupHome(); err != nil {
			return err
		}
	}
	return nil
}

func logf(s string, v ...interface{}) {
	fmt.Printf("%s\n", fmt.Sprintf(s, v...))
}
