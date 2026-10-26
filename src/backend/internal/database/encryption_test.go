package database

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/roomies/backend/internal/secret"
)

// neverAppliesUpdates wraps a migrationRunner so every UPDATE it runs
// reports zero rows affected without touching the row, standing in for a
// concurrent writer that wins the guard race on every single attempt.
type neverAppliesUpdates struct {
	*sqlx.DB
}

func (n neverAppliesUpdates) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	if strings.HasPrefix(strings.TrimSpace(query), "UPDATE") {
		return zeroRowsResult{}, nil
	}
	return n.DB.ExecContext(ctx, query, args...)
}

type zeroRowsResult struct{}

func (zeroRowsResult) LastInsertId() (int64, error) { return 0, nil }
func (zeroRowsResult) RowsAffected() (int64, error) { return 0, nil }

func TestExistingPlaintextIsEncryptedAndKeysRotate(t *testing.T) {
	db, err := Connect("sqlite://:memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := RunMigrations(db); err != nil {
		t.Fatal(err)
	}
	// Rows as an older release wrote them: plaintext, no blind index.
	if _, err := db.Exec(`INSERT INTO users (id, name, email, password_hash) VALUES ('u1', 'Ana Souza', 'Ana@Example.test', 'x');
		INSERT INTO houses (id, name) VALUES ('h1', 'Rua das Flores 12');
		INSERT INTO notes (id, house_id, author_id, title, content) VALUES ('n1', 'h1', 'u1', 'Wi-Fi', 'password hunter2');
		INSERT INTO grocery_items (id, house_id, creator_id, name) VALUES ('g1', 'h1', 'u1', 'Oat milk')`); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := EncryptExistingData(ctx, db); err != nil {
		t.Fatal(err)
	}
	var name, email, hash, content string
	if err := db.QueryRow(`SELECT u.name, u.email, u.email_hash, n.content FROM users u JOIN notes n ON n.author_id = u.id`).Scan(&name, &email, &hash, &content); err != nil {
		t.Fatal(err)
	}
	for _, stored := range []string{name, email, content} {
		if !strings.HasPrefix(stored, "enc:v1:test:") || strings.Contains(stored, "Ana") || strings.Contains(stored, "hunter2") {
			t.Fatalf("plaintext survived: %q", stored)
		}
	}
	if want, _ := secret.Index("ana@example.test"); hash != want {
		t.Fatalf("email hash = %q, want lowercase-email index", hash)
	}
	var decrypted secret.String
	if err := db.Get(&decrypted, `SELECT content FROM notes WHERE id = 'n1'`); err != nil || decrypted != "password hunter2" {
		t.Fatalf("decrypted content = %q, %v", decrypted, err)
	}

	original := secret.Active()
	defer secret.Configure(original)
	oldKey := base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{7}, 32))
	rotated, err := secret.NewKeyRing("next", base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{9}, 32)), "test:"+oldKey)
	if err != nil {
		t.Fatal(err)
	}
	secret.Configure(rotated)
	if err := RunMigrations(db); err != nil {
		t.Fatal(err)
	}
	var stillOld int
	if err := db.Get(&stillOld, `SELECT COUNT(*) FROM (
		SELECT name AS v FROM users UNION ALL SELECT email FROM users UNION ALL SELECT name FROM houses
		UNION ALL SELECT title FROM notes UNION ALL SELECT content FROM notes UNION ALL SELECT name FROM grocery_items
	) WHERE v NOT LIKE 'enc:v1:next:%'`); err != nil {
		t.Fatal(err)
	}
	if stillOld != 0 {
		t.Fatalf("%d values were not re-encrypted under the new key", stillOld)
	}
	if err := db.Get(&hash, `SELECT email_hash FROM users WHERE id = 'u1'`); err != nil {
		t.Fatal(err)
	}
	if want := rotated.Index("ana@example.test"); hash != want {
		t.Fatal("email hash was not recomputed with the new index key")
	}
	if err := db.Get(&decrypted, `SELECT name FROM grocery_items WHERE id = 'g1'`); err != nil || decrypted != "Oat milk" {
		t.Fatalf("rotated value unreadable: %q %v", decrypted, err)
	}
}

func TestLegacyMigrationTreatsEveryOldValueAsText(t *testing.T) {
	db, err := Connect("sqlite://:memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := RunMigrations(db); err != nil {
		t.Fatal(err)
	}
	// Rows an older release wrote, including text that looks like ciphertext
	// and a house.updated event that copied the house name.
	if _, err := db.Exec(`INSERT INTO users (id, name, email, password_hash) VALUES ('u1', 'Ana', 'ana@example.test', 'x');
		INSERT INTO houses (id, name) VALUES ('h1', 'Rua A');
		INSERT INTO notes (id, house_id, author_id, title, content) VALUES ('n1', 'h1', 'u1', 'enc:v1:test:notreally', 'enc:v1:example');
		INSERT INTO house_events (id, house_id, event_type, resource_type, resource_id, payload) VALUES ('e1', 'h1', 'house.updated', 'house', 'h1', '{"name":"Rua A"}');
		DELETE FROM schema_migrations WHERE version = 12`); err != nil {
		t.Fatal(err)
	}
	if err := RunMigrations(db); err != nil {
		t.Fatalf("old text shaped like ciphertext must not block startup: %v", err)
	}
	var title, content secret.String
	if err := db.QueryRow(`SELECT title, content FROM notes WHERE id = 'n1'`).Scan(&title, &content); err != nil {
		t.Fatal(err)
	}
	if title != "enc:v1:test:notreally" || content != "enc:v1:example" {
		t.Fatalf("legacy text changed: %q %q", title, content)
	}
	var payload string
	if err := db.Get(&payload, `SELECT payload FROM house_events WHERE id = 'e1'`); err != nil || strings.Contains(payload, "Rua") {
		t.Fatalf("house name left in event payload: %s %v", payload, err)
	}
}

func TestResealSkipsRowsChangedConcurrently(t *testing.T) {
	db, err := Connect("sqlite://:memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := RunMigrations(db); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO houses (id, name) VALUES ('h1', $1)`, secret.String("edited by the app")); err != nil {
		t.Fatal(err)
	}
	// The backfill read an older value; the app has since saved a new one.
	stale := staleRow{id: "h1", values: []sql.NullString{{String: "old plaintext", Valid: true}}}
	applied, err := resealRow(context.Background(), db, secret.Active(), "houses", []string{"name"}, "", "", stale, true)
	if err != nil {
		t.Fatal(err)
	}
	if applied {
		t.Fatal("guarded update reported applied, want skipped")
	}
	var name secret.String
	if err := db.Get(&name, `SELECT name FROM houses WHERE id = 'h1'`); err != nil || name != "edited by the app" {
		t.Fatalf("concurrent edit overwritten: %q %v", name, err)
	}
}

func TestResealWithRetryRecoversFromConcurrentEdit(t *testing.T) {
	db, err := Connect("sqlite://:memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := RunMigrations(db); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO users (id, name, email, password_hash) VALUES ('u1', 'Ana', 'edited@example.test', 'x')`); err != nil {
		t.Fatal(err)
	}
	ring := secret.Active()
	// The backfill read an older value for email; the row's current, actual
	// value ("edited@example.test") is what the retry's fresh read returns.
	stale := staleRow{id: "u1", values: []sql.NullString{{String: "stale@example.test", Valid: true}}}
	retryQuery := `SELECT id, email FROM users WHERE id = $1 AND (1 = 1)`

	err = resealWithRetry(context.Background(), db, ring, "users", []string{"email"}, "email", "email_hash", stale, true, retryQuery, nil)
	if err != nil {
		t.Fatalf("retry should recover once it re-reads the current value: %v", err)
	}
	var email, hash string
	if err := db.QueryRow(`SELECT email, email_hash FROM users WHERE id = 'u1'`).Scan(&email, &hash); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(email, "enc:v1:test:") {
		t.Fatalf("email not sealed: %q", email)
	}
	if want, _ := secret.Index("edited@example.test"); hash != want {
		t.Fatalf("email_hash = %q, want index of the row's real, current email", hash)
	}
	var decrypted secret.String
	if err := db.Get(&decrypted, `SELECT email FROM users WHERE id = 'u1'`); err != nil || decrypted != "edited@example.test" {
		t.Fatalf("decrypted email = %q, %v", decrypted, err)
	}
}

func TestResealWithRetryGivesUpAfterPersistentConflict(t *testing.T) {
	db, err := Connect("sqlite://:memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := RunMigrations(db); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO houses (id, name) VALUES ('h1', $1)`, secret.String("original")); err != nil {
		t.Fatal(err)
	}
	ring := secret.Active()
	stale := staleRow{id: "h1", values: []sql.NullString{{String: "old plaintext", Valid: true}}}
	// A retry query that never reflects the row's real stored value, so the
	// guarded update is doomed to keep matching nothing: it stands in for a
	// writer that keeps winning the race on every single attempt.
	retryQuery := `SELECT id, 'still wrong' AS name FROM houses WHERE id = $1`

	err = resealWithRetry(context.Background(), db, ring, "houses", []string{"name"}, "", "", stale, true, retryQuery, nil)
	if err == nil {
		t.Fatal("want an error when the guarded update never applies, got nil")
	}
	var name secret.String
	if err := db.Get(&name, `SELECT name FROM houses WHERE id = 'h1'`); err != nil || name != "original" {
		t.Fatalf("row should be untouched after exhausting retries: %q %v", name, err)
	}
}

func TestEncryptExistingDataStopsStartupWhenARowNeverSeals(t *testing.T) {
	db, err := Connect("sqlite://:memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := RunMigrations(db); err != nil {
		t.Fatal(err)
	}
	// Written as plaintext, bypassing secret.String, so EncryptExistingData's
	// non-legacy pass sees it as stale (no current-key prefix).
	if _, err := db.Exec(`INSERT INTO houses (id, name) VALUES ('h1', 'Raw Plaintext')`); err != nil {
		t.Fatal(err)
	}
	err = EncryptExistingData(context.Background(), neverAppliesUpdates{db})
	if err == nil {
		t.Fatal("want startup to stop when a row's guarded update never applies, got nil")
	}
	var name string
	if err := db.Get(&name, `SELECT name FROM houses WHERE id = 'h1'`); err != nil || name != "Raw Plaintext" {
		t.Fatalf("row should be left untouched, not partially sealed: %q %v", name, err)
	}
}
