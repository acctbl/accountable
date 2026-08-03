package migration

import (
	"context"
	"database/sql"
	"errors"
	"io/fs"

	"github.com/acctbl/accountable/internal/foundation"
	"github.com/pressly/goose/v3"
)

const migrationLockID int64 = 7_599_793_098_397_629_243

func Run(ctx context.Context, database *sql.DB, migrations fs.FS) error {
	database.SetMaxOpenConns(1)
	database.SetMaxIdleConns(1)

	var locked bool
	if err := database.QueryRowContext(ctx, `SELECT pg_try_advisory_lock($1)`, migrationLockID).Scan(&locked); err != nil || !locked {
		return errors.New("migration lock is unavailable")
	}
	defer func() {
		_, _ = database.ExecContext(context.Background(), `SELECT pg_advisory_unlock($1)`, migrationLockID)
	}()
	if _, err := database.ExecContext(ctx, `SET TIME ZONE 'UTC'`); err != nil {
		return errors.New("migration database session cannot set UTC")
	}
	var timezone string
	if err := database.QueryRowContext(ctx, `SELECT current_setting('TimeZone')`).Scan(&timezone); err != nil || timezone != "UTC" {
		return errors.New("migration database session is not UTC")
	}
	if _, err := database.ExecContext(ctx, `SET search_path TO public`); err != nil {
		return errors.New("migration database session cannot be secured")
	}

	goose.SetBaseFS(migrations)
	if err := goose.SetDialect("postgres"); err != nil {
		return errors.New("migration dialect cannot be configured")
	}
	currentVersion, err := goose.GetDBVersionContext(ctx, database)
	if err != nil {
		return errors.New("schema migration version cannot be read")
	}
	pending, err := goose.CollectMigrations(".", currentVersion, goose.MaxVersion)
	if err != nil && !errors.Is(err, goose.ErrNoMigrationFiles) {
		return errors.New("migration catalogue cannot be read")
	}
	if len(pending) == 0 {
		return validateCurrentState(ctx, database, currentVersion)
	}
	if err := markStateDirtyIfPresent(ctx, database); err != nil {
		return err
	}
	if err := goose.UpContext(ctx, database, "."); err != nil {
		return errors.New("migration failed; schema remains dirty")
	}
	version, err := goose.GetDBVersionContext(ctx, database)
	if err != nil {
		return errors.New("migrated schema version cannot be read")
	}
	if err := foundation.ValidateSchemaState(version, false); err != nil {
		return errors.New("migrations do not produce a binary-supported schema version")
	}
	result, err := database.ExecContext(ctx, `UPDATE public.accountable_schema_state SET version = $1, dirty = FALSE WHERE singleton = TRUE`, version)
	if err != nil {
		return errors.New("schema state cannot be finalized")
	}
	rows, err := result.RowsAffected()
	if err != nil || rows != 1 {
		return errors.New("schema state cannot be finalized")
	}
	return nil
}

func markStateDirtyIfPresent(ctx context.Context, database *sql.DB) error {
	var exists bool
	if err := database.QueryRowContext(ctx, `SELECT to_regclass('public.accountable_schema_state') IS NOT NULL`).Scan(&exists); err != nil {
		return errors.New("schema state cannot be read")
	}
	if !exists {
		return nil
	}
	result, err := database.ExecContext(ctx, `UPDATE public.accountable_schema_state SET dirty = TRUE WHERE singleton = TRUE`)
	if err != nil {
		return errors.New("schema state cannot be marked dirty")
	}
	rows, err := result.RowsAffected()
	if err != nil || rows != 1 {
		return errors.New("schema state cannot be marked dirty")
	}
	return nil
}

func validateCurrentState(ctx context.Context, database *sql.DB, migrationVersion int64) error {
	var version int64
	var dirty bool
	if err := database.QueryRowContext(ctx, `SELECT version, dirty FROM public.accountable_schema_state WHERE singleton = TRUE`).Scan(&version, &dirty); err != nil {
		return errors.New("schema state does not match the migration catalogue")
	}
	if version != migrationVersion {
		return errors.New("schema state does not match the migration catalogue")
	}
	if err := foundation.ValidateSchemaState(version, dirty); err != nil {
		return errors.New("schema state is not supported by this binary")
	}
	return nil
}
