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
	if migrationCount != len(migrations()) {
		t.Fatalf("expected %d recorded migrations, got %d", len(migrations()), migrationCount)
	}
}
