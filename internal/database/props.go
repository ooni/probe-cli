package database

// Database properties retreived on initialization

import (
	"github.com/ooni/probe-cli/v3/internal/model"
)

type DatabaseProps struct {
	//
	Database *Database

	//
	DatabaseNetwork *model.DatabaseNetwork

	//
	DatabaseResult *model.DatabaseResult
}
