package database

import (
	"database/sql"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/internal/bindata"
	migrate "github.com/rubenv/sql-migrate"
	"upper.io/db.v3/lib/sqlbuilder"
	"upper.io/db.v3/sqlite"
)

// RunMigrations runs the database migrations
func RunMigrations(db *sql.DB) error {
	log.Debugf("running migrations")
	migrations := &migrate.AssetMigrationSource{
		Asset:    bindata.Asset,
		AssetDir: bindata.AssetDir,
		Dir:      "data/migrations",
	}
	n, err := migrate.Exec(db, "sqlite3", migrations, migrate.Up)
	if err != nil {
		return err
	}
	log.Debugf("performed %d migrations", n)
	return nil
}

// Connect to the database
func Connect(path string) (db sqlbuilder.Database, err error) {
	settings := sqlite.ConnectionURL{
		Database: path,
		Options:  map[string]string{"_foreign_keys": "1"},
	}
	sess, err := sqlite.Open(settings)
	if err != nil {
		log.WithError(err).Error("failed to open the DB")
		return nil, err
	}

	err = RunMigrations(sess.Driver().(*sql.DB))
	if err != nil {
		db = nil
	}
	return sess, err
}
