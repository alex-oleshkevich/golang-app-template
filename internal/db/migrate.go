package db

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"

	"github.com/pressly/goose/v3"
)

const MigrationsDir = "internal/db/migrations"

//go:embed all:migrations
var migrations embed.FS

func Migrator(db *sql.DB) (*goose.Provider, error) {
	sub, err := fs.Sub(migrations, "migrations")
	if err != nil {
		return nil, err
	}
	provider, err := goose.NewProvider(goose.DialectSQLite3, db, sub)
	if errors.Is(err, goose.ErrNoMigrations) {
		return nil, fmt.Errorf("%w in %s; create one with: nisaba db new <name>", err, MigrationsDir)
	}
	return provider, err
}
