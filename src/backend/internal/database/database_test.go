package database

import (
	"testing"
)

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
	if len(names) != 3 || names[0].Name != "" || names[1].Name != "monetary_cents_columns" || names[2].Name != "reliability_platform" {
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
