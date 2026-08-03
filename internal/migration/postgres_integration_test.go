package migration_test

import (
	"context"
	"database/sql"
	"errors"
	"io/fs"
	"os"
	"testing"
	"testing/fstest"
	"time"

	"github.com/acctbl/accountable/db"
	"github.com/acctbl/accountable/internal/foundation"
	"github.com/acctbl/accountable/internal/migration"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type staticResolver struct{ dsn string }

func (r staticResolver) Resolve(context.Context, foundation.SecretRef) (foundation.SecretValue, error) {
	return foundation.NewSecretValue([]byte(r.dsn)), nil
}

func TestPostgresIntegrationRefusalMatrix(t *testing.T) {
	dsn := os.Getenv("ACCOUNTABLE_TEST_POSTGRES_DSN")
	if dsn == "" {
		if os.Getenv("ACCOUNTABLE_REQUIRE_POSTGRES") == "1" {
			t.Fatal("ACCOUNTABLE_TEST_POSTGRES_DSN is required")
		}
		t.Skip("run through task test:postgres")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	database, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open Postgres: %v", err)
	}
	defer func() { _ = database.Close() }()
	reset := func() {
		t.Helper()
		if _, err := database.ExecContext(ctx, `DROP TABLE IF EXISTS goose_db_version, accountable_schema_state`); err != nil {
			t.Fatalf("reset database: %v", err)
		}
	}
	reset()
	defer reset()

	databaseConfig := foundation.DatabaseConfig{
		DSNRef:              "database.test_dsn",
		ConnectTimeout:      5 * time.Second,
		HealthCheckInterval: time.Second,
		MaxConnections:      2,
		Timezone:            "UTC",
	}
	opened, err := foundation.OpenDatabase(ctx, databaseConfig, staticResolver{dsn: dsn})
	if !errors.Is(err, foundation.ErrSchemaBehind) || opened != nil {
		t.Fatalf("empty database open = (%v, %v), want schema behind", opened, err)
	}
	var stateTableExists bool
	if err := database.QueryRowContext(ctx, `SELECT to_regclass('public.accountable_schema_state') IS NOT NULL`).Scan(&stateTableExists); err != nil {
		t.Fatalf("check state table: %v", err)
	}
	if stateTableExists {
		t.Fatal("API database preflight mutated an empty schema")
	}

	if _, err := database.ExecContext(ctx, `SET TIME ZONE 'Europe/London'`); err != nil {
		t.Fatalf("seed non-UTC migration session: %v", err)
	}
	if err := migration.Run(ctx, database, db.Migrations()); err != nil {
		t.Fatalf("migrate empty database: %v", err)
	}
	var migrationTimezone string
	if err := database.QueryRowContext(ctx, `SELECT current_setting('TimeZone')`).Scan(&migrationTimezone); err != nil {
		t.Fatalf("read migration timezone: %v", err)
	}
	if migrationTimezone != "UTC" {
		t.Fatalf("migration timezone = %q, want UTC", migrationTimezone)
	}
	opened, err = foundation.OpenDatabase(ctx, databaseConfig, staticResolver{dsn: dsn})
	if err != nil {
		t.Fatalf("open compatible database: %v", err)
	}
	opened.Close()
	if _, err := database.ExecContext(ctx, `
CREATE TABLE migration_state_update_audit (updated_at timestamptz NOT NULL DEFAULT now());
CREATE FUNCTION audit_migration_state_update() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    INSERT INTO migration_state_update_audit DEFAULT VALUES;
    RETURN NEW;
END
$$;
CREATE TRIGGER audit_migration_state_update
AFTER UPDATE ON public.accountable_schema_state
FOR EACH ROW EXECUTE FUNCTION audit_migration_state_update()`); err != nil {
		t.Fatalf("install schema-state audit: %v", err)
	}
	if err := migration.Run(ctx, database, db.Migrations()); err != nil {
		t.Fatalf("run no-op migration: %v", err)
	}
	var stateUpdates int
	if err := database.QueryRowContext(ctx, `SELECT count(*) FROM migration_state_update_audit`).Scan(&stateUpdates); err != nil {
		t.Fatalf("count schema-state updates: %v", err)
	}
	if stateUpdates != 0 {
		t.Fatalf("no-op migration changed schema state %d times", stateUpdates)
	}
	if _, err := database.ExecContext(ctx, `
DROP TRIGGER audit_migration_state_update ON public.accountable_schema_state;
DROP FUNCTION audit_migration_state_update();
DROP TABLE migration_state_update_audit`); err != nil {
		t.Fatalf("remove schema-state audit: %v", err)
	}
	if _, err := database.ExecContext(ctx, `DELETE FROM public.accountable_schema_state`); err != nil {
		t.Fatalf("delete schema state: %v", err)
	}
	opened, err = foundation.OpenDatabase(ctx, databaseConfig, staticResolver{dsn: dsn})
	if opened != nil {
		opened.Close()
	}
	if !errors.Is(err, foundation.ErrSchemaBehind) {
		t.Fatalf("missing schema row error = %v, want schema behind", err)
	}
	if _, err := database.ExecContext(ctx, `INSERT INTO public.accountable_schema_state (singleton, version, dirty) VALUES (TRUE, $1, FALSE)`, foundation.MaximumSchemaVersion); err != nil {
		t.Fatalf("restore schema state: %v", err)
	}

	assertRefused := func(name string, version int64, dirty bool, want error) {
		t.Helper()
		if _, err := database.ExecContext(ctx, `UPDATE accountable_schema_state SET version = $1, dirty = $2`, version, dirty); err != nil {
			t.Fatalf("%s seed state: %v", name, err)
		}
		opened, err := foundation.OpenDatabase(ctx, databaseConfig, staticResolver{dsn: dsn})
		if opened != nil {
			opened.Close()
		}
		if !errors.Is(err, want) {
			t.Fatalf("%s open error = %v, want %v", name, err, want)
		}
	}
	assertRefused("behind", foundation.MinimumSchemaVersion-1, false, foundation.ErrSchemaBehind)
	assertRefused("dirty", foundation.MinimumSchemaVersion, true, foundation.ErrSchemaDirty)
	assertRefused("unknown ahead", foundation.MaximumSchemaVersion+1, false, foundation.ErrSchemaAhead)

	reset()
	if err := migration.Run(ctx, database, db.Migrations()); err != nil {
		t.Fatalf("seed schema before failed migration: %v", err)
	}
	badMigrations := fstest.MapFS{
		"00002_bad.sql": &fstest.MapFile{Data: []byte("-- +goose Up\nTHIS IS NOT SQL;\n")},
	}
	if err := migration.Run(ctx, database, fs.FS(badMigrations)); err == nil {
		t.Fatal("invalid migration succeeded")
	}
	var dirty bool
	if err := database.QueryRowContext(ctx, `SELECT dirty FROM accountable_schema_state WHERE singleton = TRUE`).Scan(&dirty); err != nil {
		t.Fatalf("read dirty state: %v", err)
	}
	if !dirty {
		t.Fatal("failed migration did not leave schema dirty")
	}
}
