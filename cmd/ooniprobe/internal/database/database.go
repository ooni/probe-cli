package database

import (
	"context"
	"database/sql"
	"embed"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	migrate "github.com/rubenv/sql-migrate"
	"github.com/upper/db/v4"
	"github.com/upper/db/v4/adapter/sqlite"
)

//go:embed migrations/*.sql
var efs embed.FS

func readAsset(path string) ([]byte, error) {
	filep, err := efs.Open(path)
	if err != nil {
		return nil, err
	}
	return netxlite.ReadAllContext(context.Background(), filep)
}

func readAssetDir(path string) ([]string, error) {
	var out []string
	lst, err := efs.ReadDir(path)
	if err != nil {
		return nil, err
	}
	for _, e := range lst {
		out = append(out, e.Name())
	}
	return out, nil
}

// RunMigrations runs the database migrations
func RunMigrations(sess *sql.DB) error {
	log.Debugf("running migrations")
	migrations := &migrate.AssetMigrationSource{
		Asset:    readAsset,
		AssetDir: readAssetDir,
		Dir:      "migrations",
	}
	n, err := migrate.Exec(sess, "sqlite3", migrations, migrate.Up)
	if err != nil {
		return err
	}
	log.Debugf("performed %d migrations", n)
	return nil
}

// Connect to the database
func Connect(path string) (sess db.Session, err error) {
	settings := sqlite.ConnectionURL{
		Database: path,
		Options:  map[string]string{"_foreign_keys": "1"},
	}
	sess, err = sqlite.Open(settings)
	if err != nil {
		log.WithError(err).Error("failed to open the DB")
		return nil, err
	}

	err = RunMigrations(sess.Driver().(*sql.DB))
	if err != nil {
		log.WithError(err).Error("failed to run DB migration")
		return nil, err
	}
	return sess, err
}
