package database

import (
	"github.com/apex/log"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3" // this is needed to load the sqlite3 driver
	"github.com/openobservatory/gooni/internal/bindata"
	migrate "github.com/rubenv/sql-migrate"
)

// RunMigrations runs the database migrations
func RunMigrations(db *sqlx.DB) error {
	log.Debugf("running migrations")
	migrations := &migrate.AssetMigrationSource{
		Asset:    bindata.Asset,
		AssetDir: bindata.AssetDir,
		Dir:      "data/migrations",
	}
	n, err := migrate.Exec(db.DB, "sqlite3", migrations, migrate.Up)
	if err != nil {
		return err
	}
	log.Debugf("performed %d migrations", n)
	return nil
}

// Connect to the database
func Connect(path string) (db *sqlx.DB, err error) {
	db, err = sqlx.Connect("sqlite3", path)
	if err != nil {
		return
	}

	err = RunMigrations(db)
	if err != nil {
		db = nil
	}
	return
}
