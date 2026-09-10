package database

import (
	"context"
	"fmt"
	"io/fs"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	goosedb "github.com/pressly/goose/v3/database"
)

// Up only, never rolls back; failure must abort startup
func Migrate(ctx context.Context, pool *pgxpool.Pool, migrations fs.FS) error {
	sqldb := stdlib.OpenDBFromPool(pool)
	defer sqldb.Close()

	provider, err := goose.NewProvider(goosedb.DialectPostgres, sqldb, migrations)
	if err != nil {
		return fmt.Errorf("database: creating migration provider: %w", err)
	}

	if _, err := provider.Up(ctx); err != nil {
		return fmt.Errorf("database: applying migrations: %w", err)
	}
	return nil
}
