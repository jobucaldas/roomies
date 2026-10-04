package database

import (
	"bytes"
	"context"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/roomies/backend/internal/secret"
)

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
