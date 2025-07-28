package store

import (
	"context"
	"database/sql"
	_ "embed"
)

//go:embed schema.sql
var ddl string

func Connect(ctx context.Context, path string) (*sql.DB, error) {
	if path == "" {
		path = ":memory:"
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	// create tables
	if _, errExec := db.ExecContext(ctx, ddl); errExec != nil {
		return nil, errExec
	}

	return db, nil
}
