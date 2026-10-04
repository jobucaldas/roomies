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

// EncryptExistingData seals plaintext left from before encryption and
// re-seals values under retired keys, refreshing email blind indexes. It runs
// on every start under the migration lock and only touches rows that need it.
func EncryptExistingData(ctx context.Context, exec migrationRunner) error {
	ring := secret.Active()
	if ring == nil {
		return secret.ErrNotConfigured
	}
	currentPrefix := "enc:v1:" + ring.CurrentID() + ":"
	for _, spec := range encryptedColumns {
		selectCols := append([]string{"id"}, spec.columns...)
		var stale []string
		for _, column := range spec.columns {
			stale = append(stale, fmt.Sprintf("(%s IS NOT NULL AND substr(%s, 1, %d) <> '%s')", column, column, len(currentPrefix), currentPrefix))
		}
		if spec.emailHash != "" {
			stale = append(stale, spec.emailHash+" IS NULL")
		}
		query := fmt.Sprintf(`SELECT %s FROM %s WHERE %s ORDER BY id LIMIT 200`, strings.Join(selectCols, ", "), spec.table, strings.Join(stale, " OR "))
		for {
			batch, err := readStaleRows(ctx, exec, query, len(spec.columns))
			if err != nil {
				return fmt.Errorf("encrypt %s: %w", spec.table, err)
			}
			if len(batch) == 0 {
				break
			}
			for _, row := range batch {
				var sets []string
				var args []any
				for i, column := range spec.columns {
					if !row.values[i].Valid {
						continue
					}
					plain, err := ring.Decrypt(row.values[i].String)
					if err != nil {
						return fmt.Errorf("encrypt %s.%s row %s: %w", spec.table, column, row.id, err)
					}
					sealed, err := ring.Encrypt(plain)
					if err != nil {
						return err
					}
					args = append(args, sealed)
					sets = append(sets, fmt.Sprintf("%s = $%d", column, len(args)))
					if column == spec.emailColumn {
						args = append(args, ring.Index(strings.ToLower(strings.TrimSpace(plain))))
						sets = append(sets, fmt.Sprintf("%s = $%d", spec.emailHash, len(args)))
					}
				}
				args = append(args, row.id)
				if _, err := exec.ExecContext(ctx, fmt.Sprintf(`UPDATE %s SET %s WHERE id = $%d`, spec.table, strings.Join(sets, ", "), len(args)), args...); err != nil {
					return fmt.Errorf("encrypt %s row %s: %w", spec.table, row.id, err)
				}
			}
		}
	}
	return nil
}

type staleRow struct {
	id     string
	values []sql.NullString
}

func readStaleRows(ctx context.Context, exec migrationRunner, query string, columns int) ([]staleRow, error) {
	rows, err := exec.QueryContext(ctx, query)
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
