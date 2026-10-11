package db

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"

	_ "modernc.org/sqlite"
)

func Open(ctx context.Context, path string) (*sql.DB, error) {
	opts := url.Values{
		"_txlock": {"immediate"},
		"_pragma": {
			"busy_timeout(5000)",
			"foreign_keys(1)",
			"journal_mode(WAL)",
			"synchronous(NORMAL)",
		},
	}
	db, err := sql.Open("sqlite", "file:"+path+"?"+opts.Encode())
	if err != nil {
		return nil, fmt.Errorf("open sqlite %s: %w", path, err)
	}
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("open sqlite %s: %w", path, err)
	}
	return db, nil
}
