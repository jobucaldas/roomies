package repository

import (
	"github.com/jmoiron/sqlx"
	"github.com/roomies/backend/internal/secret"
	"strings"
	"testing"
	"time"

	roomiesclock "github.com/roomies/backend/internal/clock"
	"github.com/roomies/backend/internal/database"
)

func setupReliabilityRepoTest(t *testing.T) (*ReliabilityRepository, *roomiesclock.FakeClock, func()) {
	t.Helper()
	db, err := database.Connect("sqlite://:memory:")
	if err != nil {
		t.Fatal(err)
	}
	if err := database.RunMigrations(db); err != nil {
		t.Fatal(err)
	}
	clk := roomiesclock.NewFake(time.Date(2025, time.January, 1, 12, 0, 0, 0, time.UTC))
	return NewReliabilityRepository(db, clk), clk, func() { _ = db.Close() }
}

func seedHouseFixture(t *testing.T, repo *ReliabilityRepository) (houseID, adminID string) {
	t.Helper()
	adminID = "user-admin"
	houseID = "house-1"
	if _, err := repo.db.Exec(`INSERT INTO users (id, name, email, password_hash, created_at) VALUES
		('user-admin', 'Admin', 'admin@example.com', 'hash', CURRENT_TIMESTAMP)`); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.db.Exec(`INSERT INTO houses (id, name, created_at) VALUES ('house-1', 'Test House', CURRENT_TIMESTAMP)`); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.db.Exec(`INSERT INTO house_members (id, house_id, user_id, role, joined_at) VALUES
		('member-admin', 'house-1', 'user-admin', 'admin', CURRENT_TIMESTAMP)`); err != nil {
		t.Fatal(err)
	}
	return houseID, adminID
}

type storedOutboxRow struct {
	Topic       string        `db:"topic"`
	Payload     secret.String `db:"payload"`
	AvailableAt time.Time     `db:"available_at"`
}

// outboxRows returns decrypted outbox rows whose payload contains every
// needle. Payloads are encrypted at rest, so SQL LIKE cannot match them.
func outboxRows(t *testing.T, db *sqlx.DB, needles ...string) []storedOutboxRow {
	t.Helper()
	var rows []storedOutboxRow
	if err := db.Select(&rows, `SELECT topic, payload, available_at FROM outbox_messages ORDER BY created_at, id`); err != nil {
		t.Fatal(err)
	}
	var matched []storedOutboxRow
	for _, row := range rows {
		ok := true
		for _, needle := range needles {
			if !strings.Contains(row.Payload.Plain(), needle) {
				ok = false
				break
			}
		}
		if ok {
			matched = append(matched, row)
		}
	}
	return matched
}

// oneOutboxPayload returns the single decrypted payload containing every needle.
func oneOutboxPayload(t *testing.T, db *sqlx.DB, needles ...string) string {
	t.Helper()
	rows := outboxRows(t, db, needles...)
	if len(rows) == 0 {
		t.Fatalf("no outbox payload contains %q", needles)
	}
	return rows[0].Payload.Plain()
}
