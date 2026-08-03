package foundation

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	// MinimumSchemaVersion and MaximumSchemaVersion are compiled into the binary.
	// Operators cannot widen this compatibility window through configuration.
	MinimumSchemaVersion int64 = 1
	MaximumSchemaVersion int64 = 1
)

var (
	ErrDatabaseUnavailable = errors.New("database is unavailable")
	ErrDatabaseTimezone    = errors.New("database session timezone is not UTC")
	ErrSchemaBehind        = errors.New("database schema is behind the supported window")
	ErrSchemaDirty         = errors.New("database schema migration is incomplete")
	ErrSchemaAhead         = errors.New("database schema is ahead of the supported window")
)

type Database struct{ pool *pgxpool.Pool }

func OpenDatabase(ctx context.Context, config DatabaseConfig, resolver SecretResolver) (*Database, error) {
	connectCtx, cancel := context.WithTimeout(ctx, config.ConnectTimeout)
	defer cancel()

	secret, err := resolver.Resolve(connectCtx, config.DSNRef)
	if err != nil {
		return nil, ErrDatabaseUnavailable
	}
	dsn := secret.Bytes()
	poolConfig, err := pgxpool.ParseConfig(string(dsn))
	clear(dsn)
	if err != nil {
		return nil, errors.New("database configuration is invalid")
	}
	poolConfig.MaxConns = config.MaxConnections
	poolConfig.MaxConnIdleTime = 5 * time.Minute
	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.HealthCheckPeriod = config.HealthCheckInterval
	poolConfig.ConnConfig.ConnectTimeout = config.ConnectTimeout
	poolConfig.AfterConnect = func(ctx context.Context, connection *pgx.Conn) error {
		if _, err := connection.Exec(ctx, `SET TIME ZONE 'UTC'`); err != nil {
			return err
		}
		_, err := connection.Exec(ctx, `SET search_path TO public`)
		return err
	}
	pool, err := pgxpool.NewWithConfig(connectCtx, poolConfig)
	if err != nil {
		return nil, ErrDatabaseUnavailable
	}
	database := &Database{pool: pool}
	if err := database.Check(connectCtx); err != nil {
		pool.Close()
		return nil, err
	}
	return database, nil
}

func (d *Database) Check(ctx context.Context) error {
	connection, err := d.pool.Acquire(ctx)
	if err != nil {
		return ErrDatabaseUnavailable
	}
	defer connection.Release()

	var timezone string
	if err := connection.QueryRow(ctx, `SELECT current_setting('TimeZone')`).Scan(&timezone); err != nil {
		return ErrDatabaseUnavailable
	}
	if timezone != "UTC" {
		return ErrDatabaseTimezone
	}

	var exists bool
	if err := connection.QueryRow(ctx, `SELECT to_regclass('public.accountable_schema_state') IS NOT NULL`).Scan(&exists); err != nil {
		return ErrDatabaseUnavailable
	}
	if !exists {
		return ErrSchemaBehind
	}
	var version int64
	var dirty bool
	if err := connection.QueryRow(ctx, `SELECT version, dirty FROM public.accountable_schema_state WHERE singleton = TRUE`).Scan(&version, &dirty); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrSchemaBehind
		}
		return ErrDatabaseUnavailable
	}
	return ValidateSchemaState(version, dirty)
}

func (d *Database) Close() { d.pool.Close() }

func ValidateSchemaState(version int64, dirty bool) error {
	if dirty {
		return ErrSchemaDirty
	}
	if version < MinimumSchemaVersion {
		return ErrSchemaBehind
	}
	if version > MaximumSchemaVersion {
		return ErrSchemaAhead
	}
	return nil
}
