package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/roomies/backend/internal/secret"
)

// encryptedColumns lists every column that holds personal data or household
// content. repository models read and write them as secret.String.
var encryptedColumns = []struct {
	table   string
	columns []string
	// emailHash names a blind-index column kept in step with the email column.
	emailHash, emailColumn string
}{
	{table: "users", columns: []string{"name", "email"}, emailHash: "email_hash", emailColumn: "email"},
	{table: "houses", columns: []string{"name"}},
	{table: "notes", columns: []string{"title", "content"}},
	{table: "expenses", columns: []string{"description", "category"}},
	{table: "house_invitations", columns: []string{"email_normalized"}, emailHash: "email_hash", emailColumn: "email_normalized"},
	{table: "audit_events", columns: []string{"metadata"}},
	{table: "grocery_items", columns: []string{"name", "quantity", "unit", "note"}},
	{table: "chores", columns: []string{"title", "description"}},
	{table: "calendar_events", columns: []string{"title", "description"}},
	{table: "chat_messages", columns: []string{"body"}},
	{table: "scheduled_house_events", columns: []string{"title"}},
	{table: "notification_subscriptions", columns: []string{"device_label"}},
	{table: "idempotency_keys", columns: []string{"response_body"}},
	{table: "outbox_messages", columns: []string{"payload"}},
}

// Length checks written for plaintext; ciphertext is longer, so they move to
// the handlers (which already enforce them).
var plaintextLengthChecks = map[string][]struct{ column, sqlite string }{
	"grocery_items": {
		{"name", " CHECK(length(trim(name)) BETWEEN 1 AND 160)"},
		{"quantity", " CHECK(length(quantity) <= 40)"},
		{"unit", " CHECK(length(unit) <= 40)"},
		{"note", " CHECK(length(note) <= 1000)"},
	},
	"chores": {
		{"title", " CHECK(length(trim(title)) BETWEEN 1 AND 160)"},
		{"description", " CHECK(length(description) <= 4000)"},
	},
	"calendar_events": {
		{"title", " CHECK(length(trim(title)) BETWEEN 1 AND 160)"},
		{"description", " CHECK(length(description) <= 4000)"},
	},
	"scheduled_house_events": {
		{"title", " CHECK(length(trim(title)) BETWEEN 1 AND 120)"},
	},
}

func encryptionSchemaMigration(driver string) func(context.Context, migrationRunner) error {
	return func(ctx context.Context, exec migrationRunner) error {
		statements := []string{
			`ALTER TABLE users ADD COLUMN email_hash TEXT`,
			`CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email_hash ON users(email_hash)`,
			`ALTER TABLE house_invitations ADD COLUMN email_hash TEXT`,
			`DROP INDEX IF EXISTS idx_house_invitations_pending_email`,
			`CREATE UNIQUE INDEX IF NOT EXISTS idx_house_invitations_pending_email_hash
				ON house_invitations(house_id, email_hash) WHERE status = 'pending'`,
		}
		if err := execStatements(statements...)(ctx, exec); err != nil {
			return err
		}
		if driver == "postgres" {
			for table, checks := range plaintextLengthChecks {
				for _, check := range checks {
					if _, err := exec.ExecContext(ctx, fmt.Sprintf(`ALTER TABLE %s DROP CONSTRAINT IF EXISTS %s_%s_check`, table, table, check.column)); err != nil {
						return err
					}
				}
			}
			return nil
		}
		return dropSQLiteChecks(ctx, exec)
	}
}

// dropSQLiteChecks removes CHECK constraints by editing the stored schema,
// the procedure SQLite documents for constraint changes that leave the
// on-disk rows untouched.
func dropSQLiteChecks(ctx context.Context, exec migrationRunner) error {
	var version int
	if err := exec.QueryRowContext(ctx, `PRAGMA schema_version`).Scan(&version); err != nil {
		return err
	}
	if _, err := exec.ExecContext(ctx, `PRAGMA writable_schema = ON`); err != nil {
		return err
	}
	for table, checks := range plaintextLengthChecks {
		var schema string
		if err := exec.QueryRowContext(ctx, `SELECT sql FROM sqlite_master WHERE type = 'table' AND name = $1`, table).Scan(&schema); err != nil {
			return fmt.Errorf("read %s schema: %w", table, err)
		}
		for _, check := range checks {
			schema = strings.Replace(schema, check.sqlite, "", 1)
		}
		if _, err := exec.ExecContext(ctx, `UPDATE sqlite_master SET sql = $1 WHERE type = 'table' AND name = $2`, schema, table); err != nil {
			return err
		}
	}
	if _, err := exec.ExecContext(ctx, fmt.Sprintf(`PRAGMA schema_version = %d`, version+1)); err != nil {
		return err
	}
	_, err := exec.ExecContext(ctx, `PRAGMA writable_schema = OFF`)
	return err
}

// encryptLegacyPlaintext is the one-time data migration that seals every
// value written before encryption existed. All of them are plaintext by
// definition, so even text that happens to look like "enc:v1:..." is
// encrypted rather than parsed. It also drops the house names that older
// house.updated events copied into their plaintext payload.
func encryptLegacyPlaintext(driver string) func(context.Context, migrationRunner) error {
	return func(ctx context.Context, exec migrationRunner) error {
		redact := `UPDATE house_events SET payload = json_remove(payload, '$.name')
			WHERE event_type = 'house.updated' AND json_valid(payload) AND json_type(payload, '$.name') IS NOT NULL`
		if driver == "postgres" {
			redact = `UPDATE house_events SET payload = (payload::jsonb - 'name')::text
				WHERE event_type = 'house.updated' AND payload::jsonb ? 'name'`
		}
		if _, err := exec.ExecContext(ctx, redact); err != nil {
			return fmt.Errorf("redact house names from events: %w", err)
		}
		return sealColumns(ctx, exec, true)
	}
}

// EncryptExistingData runs on every start under the migration lock. It
// re-seals values under retired keys (key rotation), seals plaintext column
// defaults, and refreshes email blind indexes. A prefixed value that does not
// decrypt (for example a retired key missing from DATA_ENCRYPTION_OLD_KEYS)
// stops startup instead of being treated as text.
func EncryptExistingData(ctx context.Context, exec migrationRunner) error {
	return sealColumns(ctx, exec, false)
}

// maxResealAttempts bounds how many times a row that keeps getting edited
// concurrently is re-read and re-sealed before sealColumns gives up and fails
// startup, rather than silently leaving it with a stale email_hash/old key.
const maxResealAttempts = 5

// sealColumns walks every encrypted column once, in id order. With legacy set
// every value is plaintext; otherwise only values not under the current key
// are visited. Each row is updated only if it still holds the values read; if
// a running server's concurrent edit makes the guarded update match nothing,
// the row is re-read and retried (bounded) rather than skipped, so no row is
// left with a stale email_hash or an old-key value.
func sealColumns(ctx context.Context, exec migrationRunner, legacy bool) error {
	ring := secret.Active()
	if ring == nil {
		return secret.ErrNotConfigured
	}
	currentPrefix := "enc:v1:" + ring.CurrentID() + ":"
	for _, spec := range encryptedColumns {
		selectCols := append([]string{"id"}, spec.columns...)
		filter := "1 = 1"
		args := []any{""}
		if !legacy {
			var stale []string
			for _, column := range spec.columns {
				stale = append(stale, fmt.Sprintf("(%s IS NOT NULL AND substr(%s, 1, %d) <> $2)", column, column, len(currentPrefix)))
			}
			if spec.emailHash != "" {
				stale = append(stale, spec.emailHash+" IS NULL")
			}
			filter = strings.Join(stale, " OR ")
			args = append(args, currentPrefix)
		}
		query := fmt.Sprintf(`SELECT %s FROM %s WHERE id > $1 AND (%s) ORDER BY id LIMIT 200`, strings.Join(selectCols, ", "), spec.table, filter)
		// Same staleness filter, scoped to one row: used to re-read a row
		// that a concurrent writer changed out from under the guarded update.
		retryQuery := fmt.Sprintf(`SELECT %s FROM %s WHERE id = $1 AND (%s)`, strings.Join(selectCols, ", "), spec.table, filter)
		var retryExtra []any
		if !legacy {
			retryExtra = []any{currentPrefix}
		}
		for {
			batch, err := readStaleRows(ctx, exec, query, len(spec.columns), args...)
			if err != nil {
				return fmt.Errorf("encrypt %s: %w", spec.table, err)
			}
			if len(batch) == 0 {
				break
			}
			for _, row := range batch {
				if err := resealWithRetry(ctx, exec, ring, spec.table, spec.columns, spec.emailColumn, spec.emailHash, row, legacy, retryQuery, retryExtra); err != nil {
					return err
				}
			}
			args[0] = batch[len(batch)-1].id
		}
	}
	return nil
}

// resealWithRetry seals row, and if a concurrent edit made the guarded update
// in resealRow match nothing, re-reads the row fresh and retries up to
// maxResealAttempts times. If the row was deleted in the meantime, there is
// nothing left to seal. If it keeps losing the race, sealColumns must not
// report success while the row's email_hash/value is still stale, so it
// returns an error that stops startup.
func resealWithRetry(ctx context.Context, exec migrationRunner, ring *secret.KeyRing, table string, columns []string, emailColumn, emailHash string, row staleRow, legacy bool, retryQuery string, retryExtra []any) error {
	for attempt := 1; ; attempt++ {
		applied, err := resealRow(ctx, exec, ring, table, columns, emailColumn, emailHash, row, legacy)
		if err != nil {
			return err
		}
		if applied {
			return nil
		}
		if attempt >= maxResealAttempts {
			return fmt.Errorf("encrypt %s row %s: still not sealed under the current key after %d attempts (concurrent writer)", table, row.id, maxResealAttempts)
		}
		fresh, err := readStaleRows(ctx, exec, retryQuery, len(columns), append([]any{row.id}, retryExtra...)...)
		if err != nil {
			return fmt.Errorf("encrypt %s: re-read row %s: %w", table, row.id, err)
		}
		if len(fresh) == 0 {
			return nil
		}
		row = fresh[0]
	}
}

// resealRow writes the re-sealed values of one row, guarded by the values it
// read: if the row changed in between, the update matches nothing. It
// reports whether the guarded update actually applied, so a caller can
// detect a skipped row rather than mistake it for success.
// Placeholders are numbered in the order they appear, which SQLite requires.
func resealRow(ctx context.Context, exec migrationRunner, ring *secret.KeyRing, table string, columns []string, emailColumn, emailHash string, row staleRow, legacy bool) (bool, error) {
	type assignment struct {
		column string
		value  any
	}
	var sets, guards []assignment
	var nullGuards []string
	for i, column := range columns {
		if !row.values[i].Valid {
			nullGuards = append(nullGuards, column+" IS NULL")
			continue
		}
		stored := row.values[i].String
		plain := stored
		if !legacy {
			var err error
			if plain, err = ring.Decrypt(stored); err != nil {
				return false, fmt.Errorf("encrypt %s.%s row %s: %w", table, column, row.id, err)
			}
		}
		sealed, err := ring.Encrypt(plain)
		if err != nil {
			return false, err
		}
		sets = append(sets, assignment{column, sealed})
		if column == emailColumn {
			sets = append(sets, assignment{emailHash, ring.Index(strings.ToLower(strings.TrimSpace(plain)))})
		}
		guards = append(guards, assignment{column, stored})
	}
	if len(sets) == 0 {
		return true, nil
	}
	var setSQL, whereSQL []string
	var args []any
	for _, set := range sets {
		args = append(args, set.value)
		setSQL = append(setSQL, fmt.Sprintf("%s = $%d", set.column, len(args)))
	}
	args = append(args, row.id)
	whereSQL = append(whereSQL, fmt.Sprintf("id = $%d", len(args)))
	for _, guard := range guards {
		args = append(args, guard.value)
		whereSQL = append(whereSQL, fmt.Sprintf("%s = $%d", guard.column, len(args)))
	}
	whereSQL = append(whereSQL, nullGuards...)
	query := fmt.Sprintf(`UPDATE %s SET %s WHERE %s`, table, strings.Join(setSQL, ", "), strings.Join(whereSQL, " AND "))
	result, err := exec.ExecContext(ctx, query, args...)
	if err != nil {
		return false, fmt.Errorf("encrypt %s row %s: %w", table, row.id, err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("encrypt %s row %s: rows affected: %w", table, row.id, err)
	}
	return affected > 0, nil
}

type staleRow struct {
	id     string
	values []sql.NullString
}

func readStaleRows(ctx context.Context, exec migrationRunner, query string, columns int, args ...any) ([]staleRow, error) {
	rows, err := exec.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var batch []staleRow
	for rows.Next() {
		row := staleRow{values: make([]sql.NullString, columns)}
		dest := []any{&row.id}
		for i := range row.values {
			dest = append(dest, &row.values[i])
		}
		if err := rows.Scan(dest...); err != nil {
			return nil, err
		}
		batch = append(batch, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(batch) > 0 && batch[0].id == "" {
		return nil, errors.New("row without id")
	}
	return batch, nil
}
