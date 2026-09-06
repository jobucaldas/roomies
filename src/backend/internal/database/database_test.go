package database

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
)

func TestPostgresMigrationsUpgradeLegacyMVP(t *testing.T) {
	db := connectPostgresMigrationTestDB(t)
	defer db.Close()

	if _, err := db.Exec(`CREATE TABLE schema_migrations (
		version INTEGER PRIMARY KEY,
		applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`); err != nil {
		t.Fatal(err)
	}
	for _, statement := range migrationStatementsV1 {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("seed legacy MVP schema: %v", err)
		}
	}
	if _, err := db.Exec(`INSERT INTO schema_migrations (version) VALUES (1)`); err != nil {
		t.Fatal(err)
	}
	seedLegacyFinancialRecords(t, db)

	if err := RunMigrations(db); err != nil {
		t.Fatalf("upgrade legacy MVP schema: %v", err)
	}
	if err := RunMigrations(db); err != nil {
		t.Fatalf("legacy MVP upgrade must be idempotent: %v", err)
	}

	assertCurrentPostgresMigrationState(t, db, "")
	assertFinancialRecordsPreserved(t, db)
}

func TestPostgresMigrationsUpgradeCentsAndLeaseLessSchema(t *testing.T) {
	db := connectPostgresMigrationTestDB(t)
	defer db.Close()
	ctx := context.Background()

	if _, err := db.Exec(`CREATE TABLE schema_migrations (
		version INTEGER PRIMARY KEY,
		name TEXT NOT NULL,
		applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`); err != nil {
		t.Fatal(err)
	}
	for _, statement := range migrationStatementsV1 {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("seed v1 schema: %v", err)
		}
	}
	if _, err := db.Exec(`INSERT INTO schema_migrations (version, name) VALUES (1, 'base_schema')`); err != nil {
		t.Fatal(err)
	}
	seedLegacyFinancialRecords(t, db)
	if err := migrations("postgres")[1].up(ctx, db); err != nil {
		t.Fatalf("apply historical cents migration: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO schema_migrations (version, name) VALUES (2, 'monetary_cents_columns')`); err != nil {
		t.Fatal(err)
	}
	if err := migrations("postgres")[2].up(ctx, db); err != nil {
		t.Fatalf("apply historical reliability migration: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO schema_migrations (version, name) VALUES (3, 'reliability_platform')`); err != nil {
		t.Fatal(err)
	}
	var leaseGenerationBefore bool
	if err := db.Get(&leaseGenerationBefore, `SELECT EXISTS (
		SELECT 1 FROM information_schema.columns
		WHERE table_schema = current_schema() AND table_name = 'durable_jobs' AND column_name = 'lease_generation'
	)`); err != nil {
		t.Fatal(err)
	}
	if leaseGenerationBefore {
		t.Fatal("historical v3 fixture unexpectedly has durable_jobs.lease_generation")
	}

	legacyResponse := `{"id":"invite-1","status":"pending","manual_acceptance_url":"https://roomies.test/accept-invitation?token=legacy-postgres-token"}`
	unrelatedResponse := `{"id":"other-1","manual_acceptance_url":"https://roomies.test/other?token=keep-me"}`
	if _, err := db.Exec(`INSERT INTO idempotency_keys
		(id, user_id, idempotency_key, method, path, body_hash, state, status_code, response_body, response_content_type)
		VALUES ('invite-key', 'user-1', 'invite-key', 'POST', '/api/houses/house-1/invites', 'hash', 'completed', 201, $1, 'application/json'),
		('other-key', 'user-1', 'other-key', 'POST', '/api/houses/house-1/notes', 'hash', 'completed', 201, $2, 'application/json')`, legacyResponse, unrelatedResponse); err != nil {
		t.Fatalf("seed legacy idempotency responses: %v", err)
	}
	if err := RunMigrations(db); err != nil {
		t.Fatalf("upgrade cents and lease-less schema: %v", err)
	}
	if err := RunMigrations(db); err != nil {
		t.Fatalf("cents and lease-less upgrade must be idempotent: %v", err)
	}

	assertCurrentPostgresMigrationState(t, db, "base_schema")
	assertFinancialRecordsPreserved(t, db)
	var stored, unrelated string
	if err := db.QueryRow(`SELECT response_body FROM idempotency_keys WHERE id = 'invite-key'`).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if stored == legacyResponse || strings.Contains(stored, "legacy-postgres-token") || strings.Contains(stored, "manual_acceptance_url") {
		t.Fatalf("legacy invitation token was not redacted: %s", stored)
	}
	if err := db.QueryRow(`SELECT response_body FROM idempotency_keys WHERE id = 'other-key'`).Scan(&unrelated); err != nil {
		t.Fatal(err)
	}
	if unrelated != unrelatedResponse {
		t.Fatalf("unrelated idempotency response changed: %s", unrelated)
	}
}

func connectPostgresMigrationTestDB(t *testing.T) *sqlx.DB {
	t.Helper()
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is required for PostgreSQL migration compatibility tests")
	}
	if driver, _ := parseURL(databaseURL); driver != "postgres" {
		t.Skip("TEST_DATABASE_URL is not PostgreSQL")
	}
	base, err := Connect(databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	schema := fmt.Sprintf("roomies_migration_test_%d", time.Now().UnixNano())
	if _, err := base.Exec(`CREATE SCHEMA ` + schema); err != nil {
		base.Close()
		t.Fatalf("create disposable PostgreSQL schema: %v", err)
	}
	schemaURL, err := postgresSchemaURL(databaseURL, schema)
	if err != nil {
		base.Close()
		t.Fatalf("configure disposable PostgreSQL schema: %v", err)
	}
	db, err := Connect(schemaURL)
	if err != nil {
		_, _ = base.Exec(`DROP SCHEMA ` + schema + ` CASCADE`)
		base.Close()
		t.Fatalf("connect disposable PostgreSQL schema: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
		if _, err := base.Exec(`DROP SCHEMA ` + schema + ` CASCADE`); err != nil {
			t.Errorf("drop disposable PostgreSQL schema: %v", err)
		}
		_ = base.Close()
	})
	return db
}

func postgresSchemaURL(databaseURL, schema string) (string, error) {
	u, err := url.Parse(databaseURL)
	if err != nil {
		return "", err
	}
	query := u.Query()
	options := strings.TrimSpace(query.Get("options"))
	if options != "" {
		options += " "
	}
	query.Set("options", options+"-csearch_path="+schema)
	u.RawQuery = query.Encode()
	return u.String(), nil
}

func seedLegacyFinancialRecords(t *testing.T, db *sqlx.DB) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO users (id, name, email, password_hash) VALUES
		('user-1', 'Alice', 'alice@example.test', 'hash-a'),
		('user-2', 'Bob', 'bob@example.test', 'hash-b')`); err != nil {
		t.Fatalf("seed users: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO houses (id, name) VALUES ('house-1', 'Legacy House')`); err != nil {
		t.Fatalf("seed house: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO house_members (id, house_id, user_id, role) VALUES
		('member-1', 'house-1', 'user-1', 'admin'),
		('member-2', 'house-1', 'user-2', 'member')`); err != nil {
		t.Fatalf("seed members: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO expenses (id, house_id, payer_id, amount, description, category, date, visibility) VALUES
		('expense-1', 'house-1', 'user-1', 12.34, 'Groceries', 'Food', '2025-01-15', 'shared')`); err != nil {
		t.Fatalf("seed expense: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO expense_splits (id, expense_id, user_id, share_amount) VALUES
		('split-1', 'expense-1', 'user-1', 6.17),
		('split-2', 'expense-1', 'user-2', 6.17)`); err != nil {
		t.Fatalf("seed splits: %v", err)
	}
}

func assertCurrentPostgresMigrationState(t *testing.T, db *sqlx.DB, legacyName string) {
	t.Helper()
	var count int
	if err := db.Get(&count, "SELECT COUNT(*) FROM schema_migrations"); err != nil {
		t.Fatal(err)
	}
	if count != len(migrations("postgres")) {
		t.Fatalf("expected %d recorded migrations, got %d", len(migrations("postgres")), count)
	}
	var ledger []struct {
		Version int    `db:"version"`
		Name    string `db:"name"`
	}
	if err := db.Select(&ledger, "SELECT version, name FROM schema_migrations ORDER BY version"); err != nil {
		t.Fatal(err)
	}
	expected := []struct {
		Version int
		Name    string
	}{{1, legacyName}, {2, "monetary_cents_columns"}, {3, "reliability_platform"}, {4, "durable_job_lease_generation"}, {5, "redact_invitation_idempotency_tokens"}}
	if len(ledger) != len(expected) {
		t.Fatalf("unexpected migration ledger: %#v", ledger)
	}
	for i := range expected {
		if ledger[i].Version != expected[i].Version || ledger[i].Name != expected[i].Name {
			t.Fatalf("unexpected migration ledger: %#v", ledger)
		}
	}
	var leaseGeneration bool
	if err := db.Get(&leaseGeneration, `SELECT EXISTS (
		SELECT 1 FROM information_schema.columns
		WHERE table_schema = current_schema() AND table_name = 'durable_jobs' AND column_name = 'lease_generation'
	)`); err != nil {
		t.Fatal(err)
	}
	if !leaseGeneration {
		t.Fatal("durable_jobs.lease_generation was not added")
	}
}

func assertFinancialRecordsPreserved(t *testing.T, db *sqlx.DB) {
	t.Helper()
	var expenseCount, splitCount int
	if err := db.Get(&expenseCount, `SELECT COUNT(*) FROM expenses WHERE id = 'expense-1' AND description = 'Groceries' AND category = 'Food'`); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&splitCount, `SELECT COUNT(*) FROM expense_splits WHERE expense_id = 'expense-1'`); err != nil {
		t.Fatal(err)
	}
	if expenseCount != 1 || splitCount != 2 {
		t.Fatalf("financial records were not preserved: expenses=%d splits=%d", expenseCount, splitCount)
	}
	var amountCents int64
	if err := db.Get(&amountCents, `SELECT amount_cents FROM expenses WHERE id = 'expense-1'`); err != nil {
		t.Fatal(err)
	}
	if amountCents != 1234 {
		t.Fatalf("expected preserved amount_cents 1234, got %d", amountCents)
	}
	var splitCents []int64
	if err := db.Select(&splitCents, `SELECT share_amount_cents FROM expense_splits WHERE expense_id = 'expense-1' ORDER BY id`); err != nil {
		t.Fatal(err)
	}
	if len(splitCents) != 2 || splitCents[0] != 617 || splitCents[1] != 617 {
		t.Fatalf("expected preserved split cents [617 617], got %v", splitCents)
	}
}

func TestSQLiteForeignKeysAndMigrations(t *testing.T) {
	db, err := Connect("sqlite://:memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := RunMigrations(db); err != nil {
		t.Fatal(err)
	}
	if err := RunMigrations(db); err != nil {
		t.Fatalf("migrations must be idempotent: %v", err)
	}

	var foreignKeys int
	if err := db.Get(&foreignKeys, "PRAGMA foreign_keys"); err != nil {
		t.Fatal(err)
	}
	if foreignKeys != 1 {
		t.Fatalf("expected SQLite foreign keys enabled, got %d", foreignKeys)
	}
	if _, err := db.Exec(`INSERT INTO house_members (id, house_id, user_id, role)
		VALUES ('member', 'missing-house', 'missing-user', 'member')`); err == nil {
		t.Fatal("expected invalid foreign-key insert to fail")
	}

	var migrationCount int
	if err := db.Get(&migrationCount, "SELECT COUNT(*) FROM schema_migrations"); err != nil {
		t.Fatal(err)
	}
	expectedMigrations := len(migrations(db.DriverName()))
	if migrationCount != expectedMigrations {
		t.Fatalf("expected %d recorded migrations, got %d", expectedMigrations, migrationCount)
	}
}

func TestSQLiteMigrationsUpgradeLegacySchema(t *testing.T) {
	db, err := Connect("sqlite://:memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	for _, statement := range migrationStatementsV1 {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("seed legacy schema: %v", err)
		}
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY,
		applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`); err != nil {
		t.Fatalf("create legacy migration ledger: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO schema_migrations (version) VALUES (1)`); err != nil {
		t.Fatalf("seed legacy migration ledger: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO users (id, name, email, password_hash) VALUES
		('user-1', 'Alice', 'alice@example.com', 'hash-a'),
		('user-2', 'Bob', 'bob@example.com', 'hash-b')`); err != nil {
		t.Fatalf("seed users: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO houses (id, name) VALUES ('house-1', 'Legacy House')`); err != nil {
		t.Fatalf("seed house: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO house_members (id, house_id, user_id, role) VALUES
		('member-1', 'house-1', 'user-1', 'admin'),
		('member-2', 'house-1', 'user-2', 'member')`); err != nil {
		t.Fatalf("seed members: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO expenses (id, house_id, payer_id, amount, description, category, date, visibility) VALUES
		('expense-1', 'house-1', 'user-1', 12.34, 'Groceries', 'Food', '2025-01-15', 'shared')`); err != nil {
		t.Fatalf("seed expense: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO expense_splits (id, expense_id, user_id, share_amount) VALUES
		('split-1', 'expense-1', 'user-1', 6.17),
		('split-2', 'expense-1', 'user-2', 6.17)`); err != nil {
		t.Fatalf("seed splits: %v", err)
	}

	if err := RunMigrations(db); err != nil {
		t.Fatalf("upgrade legacy schema: %v", err)
	}
	if err := RunMigrations(db); err != nil {
		t.Fatalf("legacy schema must remain idempotent after upgrade: %v", err)
	}

	var migrationCount int
	if err := db.Get(&migrationCount, "SELECT COUNT(*) FROM schema_migrations"); err != nil {
		t.Fatal(err)
	}
	expectedMigrations := len(migrations(db.DriverName()))
	if migrationCount != expectedMigrations {
		t.Fatalf("expected %d recorded migrations, got %d", expectedMigrations, migrationCount)
	}

	var names []struct {
		Name string `db:"name"`
	}
	if err := db.Select(&names, "SELECT name FROM schema_migrations ORDER BY version"); err != nil {
		t.Fatal(err)
	}
	if len(names) != 5 || names[0].Name != "" || names[1].Name != "monetary_cents_columns" || names[2].Name != "reliability_platform" || names[3].Name != "durable_job_lease_generation" || names[4].Name != "redact_invitation_idempotency_tokens" {
		t.Fatalf("unexpected migration ledger names: %#v", names)
	}

	var amountCents int64
	if err := db.Get(&amountCents, "SELECT amount_cents FROM expenses WHERE id = 'expense-1'"); err != nil {
		t.Fatal(err)
	}
	if amountCents != 1234 {
		t.Fatalf("expected 1234 amount_cents, got %d", amountCents)
	}

	var splitCents []struct {
		ShareAmountCents int64 `db:"share_amount_cents"`
	}
	if err := db.Select(&splitCents, "SELECT share_amount_cents FROM expense_splits WHERE expense_id = 'expense-1' ORDER BY id"); err != nil {
		t.Fatal(err)
	}
	if len(splitCents) != 2 || splitCents[0].ShareAmountCents != 617 || splitCents[1].ShareAmountCents != 617 {
		t.Fatalf("expected split cents backfill, got %#v", splitCents)
	}
}

func TestSQLiteMigrationRedactsLegacyInvitationIdempotencyToken(t *testing.T) {
	db, err := Connect("sqlite://:memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	for _, migration := range migrations(db.DriverName())[:4] {
		if err := migration.up(ctx, db); err != nil {
			t.Fatalf("apply pre-redaction migration %d: %v", migration.version, err)
		}
	}
	if _, err := db.Exec(`CREATE TABLE schema_migrations (version INTEGER PRIMARY KEY, name TEXT NOT NULL, applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP)`); err != nil {
		t.Fatal(err)
	}
	for _, migration := range migrations(db.DriverName())[:4] {
		if _, err := db.Exec(`INSERT INTO schema_migrations (version, name) VALUES ($1, $2)`, migration.version, migration.name); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec(`INSERT INTO users (id, name, email, password_hash) VALUES ('user-1', 'User', 'user@example.test', 'hash')`); err != nil {
		t.Fatal(err)
	}
	legacyResponse := `{"id":"invite-1","status":"pending","manual_acceptance_url":"https://roomies.test/accept-invitation?token=legacy-bearer-token"}`
	if _, err := db.Exec(`INSERT INTO idempotency_keys (id, user_id, idempotency_key, method, path, body_hash, state, status_code, response_body, response_content_type)
		VALUES ('key-1', 'user-1', 'key-1', 'POST', '/api/houses/house-1/invites', 'hash', 'completed', 201, $1, 'application/json')`, legacyResponse); err != nil {
		t.Fatal(err)
	}
	if err := RunMigrations(db); err != nil {
		t.Fatal(err)
	}
	var stored string
	if err := db.Get(&stored, `SELECT response_body FROM idempotency_keys WHERE id = 'key-1'`); err != nil {
		t.Fatal(err)
	}
	if stored == legacyResponse || strings.Contains(stored, "legacy-bearer-token") {
		t.Fatalf("legacy idempotency token was not redacted: %s", stored)
	}
}

func TestSQLiteReliabilityTablesAreCreated(t *testing.T) {
	db, err := Connect("sqlite://:memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := RunMigrations(db); err != nil {
		t.Fatal(err)
	}

	tables := []string{"house_invitations", "idempotency_keys", "outbox_messages", "durable_jobs", "audit_events", "house_events"}
	for _, table := range tables {
		var count int
		if err := db.Get(&count, `SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = $1`, table); err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Fatalf("expected table %s to exist", table)
		}
	}
}
