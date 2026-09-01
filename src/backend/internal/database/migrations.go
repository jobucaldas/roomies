package database

import (
	"context"
	"database/sql"
	"fmt"
)

type migrationRunner interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

type schemaMigration struct {
	version int
	name    string
	up      func(context.Context, migrationRunner) error
}

func migrations() []schemaMigration {
	return []schemaMigration{
		{
			version: 1,
			name:    "base_schema",
			up:      execStatements(migrationStatementsV1...),
		},
		{
			version: 2,
			name:    "monetary_cents_columns",
			up:      execStatements(migrationStatementsV2...),
		},
	}
}

func execStatements(statements ...string) func(context.Context, migrationRunner) error {
	return func(ctx context.Context, exec migrationRunner) error {
		for _, statement := range statements {
			if _, err := exec.ExecContext(ctx, statement); err != nil {
				return err
			}
		}
		return nil
	}
}

func ensureMigrationLedgerCompatibility(ctx context.Context, exec migrationRunner, driver string) error {
	hasName, err := migrationLedgerHasColumn(ctx, exec, driver, "name")
	if err != nil {
		return err
	}
	if hasName {
		return nil
	}
	if _, err := exec.ExecContext(ctx, `ALTER TABLE schema_migrations ADD COLUMN name TEXT NOT NULL DEFAULT ''`); err != nil {
		return fmt.Errorf("add migration ledger name column: %w", err)
	}
	return nil
}

func migrationLedgerHasColumn(ctx context.Context, exec migrationRunner, driver, column string) (bool, error) {
	switch driver {
	case "sqlite3":
		rows, err := exec.QueryContext(ctx, `PRAGMA table_info(schema_migrations)`)
		if err != nil {
			return false, fmt.Errorf("inspect migration ledger: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			var cid int
			var name, colType string
			var notNull int
			var defaultValue sql.NullString
			var pk int
			if err := rows.Scan(&cid, &name, &colType, &notNull, &defaultValue, &pk); err != nil {
				return false, fmt.Errorf("inspect migration ledger: %w", err)
			}
			if name == column {
				return true, nil
			}
		}
		if err := rows.Err(); err != nil {
			return false, fmt.Errorf("inspect migration ledger: %w", err)
		}
		return false, nil
	default:
		var exists bool
		if err := exec.QueryRowContext(ctx, `SELECT EXISTS (
			SELECT 1
			FROM information_schema.columns
			WHERE table_schema = current_schema()
				AND table_name = 'schema_migrations'
				AND column_name = $1
		)`, column).Scan(&exists); err != nil {
			return false, fmt.Errorf("inspect migration ledger: %w", err)
		}
		return exists, nil
	}
}

var migrationStatementsV1 = []string{
	`CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		email TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`,
	`CREATE TABLE IF NOT EXISTS houses (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`,
	`CREATE TABLE IF NOT EXISTS house_members (
		id TEXT PRIMARY KEY,
		house_id TEXT NOT NULL REFERENCES houses(id) ON DELETE CASCADE,
		user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		role TEXT NOT NULL CHECK(role IN ('admin','member','monitor')),
		joined_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(house_id, user_id)
	)`,
	`CREATE TABLE IF NOT EXISTS expenses (
		id TEXT PRIMARY KEY,
		house_id TEXT NOT NULL REFERENCES houses(id) ON DELETE CASCADE,
		payer_id TEXT NOT NULL REFERENCES users(id),
		amount DECIMAL(10,2) NOT NULL CHECK(amount > 0 AND amount <= 99999999.99),
		description TEXT NOT NULL CHECK(length(trim(description)) > 0),
		category TEXT DEFAULT '',
		date DATE NOT NULL,
		visibility TEXT NOT NULL DEFAULT 'shared' CHECK(visibility IN ('shared','private')),
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`,
	`CREATE TABLE IF NOT EXISTS expense_visibility (
		expense_id TEXT NOT NULL REFERENCES expenses(id) ON DELETE CASCADE,
		user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		PRIMARY KEY(expense_id, user_id)
	)`,
	`CREATE TABLE IF NOT EXISTS expense_splits (
		id TEXT PRIMARY KEY,
		expense_id TEXT NOT NULL REFERENCES expenses(id) ON DELETE CASCADE,
		user_id TEXT NOT NULL REFERENCES users(id),
		share_amount DECIMAL(10,2) NOT NULL CHECK(share_amount >= 0 AND share_amount <= 99999999.99),
		UNIQUE(expense_id, user_id)
	)`,
	`CREATE TABLE IF NOT EXISTS notes (
		id TEXT PRIMARY KEY,
		house_id TEXT NOT NULL REFERENCES houses(id) ON DELETE CASCADE,
		author_id TEXT NOT NULL REFERENCES users(id),
		title TEXT NOT NULL,
		content TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`,
	`CREATE INDEX IF NOT EXISTS idx_expenses_house ON expenses(house_id)`,
	`CREATE INDEX IF NOT EXISTS idx_notes_house ON notes(house_id)`,
	`CREATE INDEX IF NOT EXISTS idx_house_members_house ON house_members(house_id)`,
	`CREATE INDEX IF NOT EXISTS idx_house_members_user ON house_members(user_id)`,
	`CREATE INDEX IF NOT EXISTS idx_expense_visibility_user ON expense_visibility(user_id)`,
	`CREATE INDEX IF NOT EXISTS idx_expense_splits_expense ON expense_splits(expense_id)`,
	`CREATE INDEX IF NOT EXISTS idx_expense_splits_user ON expense_splits(user_id)`,
}

var migrationStatementsV2 = []string{
	`ALTER TABLE expenses ADD COLUMN amount_cents BIGINT NOT NULL DEFAULT 1 CHECK(amount_cents > 0 AND amount_cents <= 9999999999)`,
	`ALTER TABLE expense_splits ADD COLUMN share_amount_cents BIGINT NOT NULL DEFAULT 0 CHECK(share_amount_cents >= 0 AND share_amount_cents <= 9999999999)`,
	`UPDATE expenses
		SET amount_cents = CAST(ROUND(amount * 100) AS BIGINT),
		    amount = CAST(ROUND(amount * 100) AS BIGINT) / 100.0`,
	`UPDATE expense_splits
		SET share_amount_cents = CAST(ROUND(share_amount * 100) AS BIGINT),
		    share_amount = CAST(ROUND(share_amount * 100) AS BIGINT) / 100.0`,
}
