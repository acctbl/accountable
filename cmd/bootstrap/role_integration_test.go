package main

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/acctbl/accountable/internal/platform/secret"
	"github.com/jackc/pgx/v5"
)

func TestEnsureLoginRolePostgresIntegration(t *testing.T) {
	dsn := os.Getenv("ACCOUNTABLE_TEST_POSTGRES_DSN")
	if dsn == "" {
		if os.Getenv("ACCOUNTABLE_REQUIRE_POSTGRES") == "1" {
			t.Fatal("ACCOUNTABLE_TEST_POSTGRES_DSN is required")
		}
		t.Skip("run through task test:postgres")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	connection, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("connect Postgres: %v", err)
	}
	defer func() { _ = connection.Close(context.Background()) }()

	const role = "accountable_role_proof"
	dropRole := func() {
		_, _ = connection.Exec(context.Background(), "DROP ROLE IF EXISTS "+pgx.Identifier{role}.Sanitize())
	}
	dropRole()
	t.Cleanup(dropRole)

	password := secret.NewSecretValue([]byte("proof-password-with-'quote-and-\\backslash"))
	if err := ensureLoginRole(ctx, connection, role, password); err != nil {
		t.Fatalf("create login role: %v", err)
	}

	var canLogin bool
	if err := connection.QueryRow(
		ctx,
		"SELECT rolcanlogin FROM pg_roles WHERE rolname = $1",
		role,
	).Scan(&canLogin); err != nil {
		t.Fatalf("read role: %v", err)
	}
	if !canLogin {
		t.Fatal("role must be able to log in")
	}

	if err := ensureLoginRole(ctx, connection, role, password); err != nil {
		t.Fatalf("reapply login role: %v", err)
	}
}
