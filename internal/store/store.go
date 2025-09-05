//go:generate go tool sqlc generate -f .sqlc.yaml
package store

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"net/http"
	"runtime"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/httpfs"
)

// MigrationAction is the type of migration to perform.
type MigrationAction int

const (
	// MigrateUp Fully upgrades the schema.
	MigrateUp MigrationAction = iota
	// MigrateDn Fully downgrades the schema.
	MigrateDn
	// MigrateUpOne Upgrade the schema by one revision.
	MigrateUpOne
	// MigrateDownOne Downgrade the schema by one revision.
	MigrateDownOne
)

var (
	//go:embed migrations
	migrations embed.FS

	ErrDBConnect = errors.New("db connect error")
	ErrMigrate   = errors.New("failed to migrate db schema")
)

func configureConnection(ctx context.Context, connection *sql.DB) error {
	parallelism := min(8, max(2, runtime.GOMAXPROCS(0)))
	connection.SetMaxOpenConns(parallelism)
	connection.SetMaxIdleConns(parallelism)
	connection.SetConnMaxLifetime(0)
	connection.SetConnMaxIdleTime(0)

	pragmas := []string{
		"PRAGMA foreign_keys = ON",
		"PRAGMA busy_timeout = 10000",
		"PRAGMA journal_mode=WAL",
		"PRAGMA main.synchronous = NORMAL",
		"PRAGMA main.cache_size = -32768",
	}
	for _, pragma := range pragmas {
		if _, errPragma := connection.ExecContext(ctx, pragma); errPragma != nil {
			return errors.Join(errPragma, ErrDBConnect)
		}
	}

	return nil
}

func Open(ctx context.Context, path string, autoMigrate bool) (*sql.DB, error) {
	if path == "" {
		path = ":memory:"
	}

	path += "?cache=private"
	connection, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, errors.Join(err, ErrDBConnect)
	}

	if errConfig := configureConnection(ctx, connection); errConfig != nil {
		return nil, errConfig
	}

	pingCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := connection.PingContext(pingCtx); err != nil {
		connection.Close()

		return nil, errors.Join(err, ErrDBConnect)
	}

	if autoMigrate {
		if errMigrate := Migrate(connection, MigrateUp); errMigrate != nil {
			return nil, errors.Join(errMigrate, ErrDBConnect)
		}
	}

	return connection, nil
}

func Migrate(conn *sql.DB, action MigrationAction) error {
	driver, errDriver := sqlite.WithInstance(conn, &sqlite.Config{})
	if errDriver != nil {
		return errors.Join(errDriver, ErrMigrate)
	}

	source, errHTTPFS := httpfs.New(http.FS(migrations), "migrations")
	if errHTTPFS != nil {
		return errors.Join(errHTTPFS, ErrMigrate)
	}

	migrator, errMigrateInstance := migrate.NewWithInstance("iofs", source, "pgx", driver)
	if errMigrateInstance != nil {
		return errors.Join(errMigrateInstance, ErrMigrate)
	}

	var errMigration error

	switch action {
	case MigrateUpOne:
		errMigration = migrator.Steps(1)
	case MigrateDn:
		errMigration = migrator.Down()
	case MigrateDownOne:
		errMigration = migrator.Steps(-1)
	case MigrateUp:
		fallthrough
	default:
		errMigration = migrator.Up()
	}

	if errMigration != nil && !errors.Is(errMigration, migrate.ErrNoChange) {
		return errors.Join(errMigration, ErrMigrate)
	}

	return nil
}
