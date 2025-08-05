package store

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
)

//go:embed schema.sql
var ddl string

var errDBConnect = errors.New("db connect error")

func Connect(ctx context.Context, path string) (*sql.DB, error) {
	if path == "" {
		path = ":memory:"
	}

	connection, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, errors.Join(err, errDBConnect)
	}

	// create tables
	if _, errExec := connection.ExecContext(ctx, ddl); errExec != nil {
		return nil, errors.Join(errExec, errDBConnect)
	}

	return connection, nil
}
