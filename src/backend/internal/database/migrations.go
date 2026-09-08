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

func migrations(driver string) []schemaMigration {
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
		{
			version: 3,
			name:    "reliability_platform",
			up:      execStatements(reliabilityStatements(driver)...),
		},
		{
			version: 4,
			name:    "durable_job_lease_generation",
			up:      execStatements(`ALTER TABLE durable_jobs ADD COLUMN lease_generation BIGINT NOT NULL DEFAULT 0`),
		},
		{
			version: 5,
			name:    "redact_invitation_idempotency_tokens",
			up:      execStatements(redactInvitationIdempotencyTokensStatement(driver)),
		},
		{
			version: 6,
			name:    "notifications_and_scheduled_events",
			up:      execStatements(notificationStatements()...),
		},
		{
			version: 7,
			name:    "house_scoped_notification_capabilities",
			up:      execStatements(`DROP INDEX IF EXISTS idx_notification_subscriptions_active_identity`),
		},
		{
			version: 8,
			name:    "household_domains",
			up:      execStatements(householdStatements(driver)...),
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

// redactInvitationIdempotencyTokensStatement removes legacy one-time invitation URLs
// from completed response snapshots. The path constraint leaves unrelated idempotency
// responses untouched; new responses are redacted before they reach this table.
func redactInvitationIdempotencyTokensStatement(driver string) string {
	if driver == "postgres" {
		return `UPDATE idempotency_keys
			SET response_body = (response_body::jsonb - 'manual_acceptance_url')::text
			WHERE method = 'POST' AND path LIKE '/api/houses/%/invites'
				AND response_body IS NOT NULL
				AND response_body::jsonb ? 'manual_acceptance_url'`
	}
	return `UPDATE idempotency_keys
		SET response_body = json_remove(response_body, '$.manual_acceptance_url')
		WHERE method = 'POST' AND path LIKE '/api/houses/%/invites'
			AND response_body IS NOT NULL
			AND json_valid(response_body)
			AND json_type(response_body, '$.manual_acceptance_url') IS NOT NULL`
}

func reliabilityStatements(driver string) []string {
	houseEventCursor := `INTEGER PRIMARY KEY AUTOINCREMENT`
	if driver == "postgres" {
		houseEventCursor = `BIGSERIAL PRIMARY KEY`
	}
	return []string{
		`CREATE TABLE IF NOT EXISTS house_invitations (
			id TEXT PRIMARY KEY,
			house_id TEXT NOT NULL REFERENCES houses(id) ON DELETE CASCADE,
			email_normalized TEXT NOT NULL,
			role TEXT NOT NULL CHECK(role IN ('admin','member','monitor')),
			token_hash TEXT NOT NULL UNIQUE,
			status TEXT NOT NULL CHECK(status IN ('pending','accepted','revoked','expired')) DEFAULT 'pending',
			created_by TEXT NOT NULL REFERENCES users(id),
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			expires_at TIMESTAMP NOT NULL,
			accepted_by TEXT REFERENCES users(id),
			accepted_at TIMESTAMP,
			revoked_by TEXT REFERENCES users(id),
			revoked_at TIMESTAMP
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_house_invitations_pending_email
			ON house_invitations(house_id, email_normalized) WHERE status = 'pending'`,
		`CREATE INDEX IF NOT EXISTS idx_house_invitations_house ON house_invitations(house_id, created_at DESC)`,
		`CREATE TABLE IF NOT EXISTS idempotency_keys (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			idempotency_key TEXT NOT NULL,
			method TEXT NOT NULL,
			path TEXT NOT NULL,
			body_hash TEXT NOT NULL,
			state TEXT NOT NULL CHECK(state IN ('processing','completed')),
			status_code INTEGER,
			response_body TEXT,
			response_content_type TEXT,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			completed_at TIMESTAMP,
			UNIQUE(user_id, idempotency_key)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_idempotency_keys_lookup ON idempotency_keys(user_id, idempotency_key)`,
		`CREATE TABLE IF NOT EXISTS outbox_messages (
			id TEXT PRIMARY KEY,
			topic TEXT NOT NULL,
			dedupe_key TEXT NOT NULL UNIQUE,
			payload TEXT NOT NULL,
			status TEXT NOT NULL CHECK(status IN ('pending','dispatched','dead_lettered')) DEFAULT 'pending',
			available_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			dispatched_at TIMESTAMP,
			last_error TEXT,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_outbox_messages_status ON outbox_messages(status, available_at)`,
		`CREATE TABLE IF NOT EXISTS durable_jobs (
			id TEXT PRIMARY KEY,
			kind TEXT NOT NULL,
			dedupe_key TEXT NOT NULL UNIQUE,
			payload TEXT NOT NULL,
			available_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			lease_owner TEXT,
			lease_expires_at TIMESTAMP,
			attempts INTEGER NOT NULL DEFAULT 0,
			max_attempts INTEGER NOT NULL DEFAULT 5,
			last_error TEXT,
			dead_lettered_at TIMESTAMP,
			completed_at TIMESTAMP,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_durable_jobs_dispatch ON durable_jobs(completed_at, dead_lettered_at, available_at)`,
		`CREATE TABLE IF NOT EXISTS audit_events (
			id TEXT PRIMARY KEY,
			actor_id TEXT REFERENCES users(id),
			house_id TEXT REFERENCES houses(id) ON DELETE CASCADE,
			action TEXT NOT NULL,
			target_type TEXT NOT NULL,
			target_id TEXT NOT NULL,
			metadata TEXT NOT NULL DEFAULT '{}',
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_events_house ON audit_events(house_id, created_at DESC)`,
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS house_events (
			cursor %s,
			id TEXT NOT NULL UNIQUE,
			house_id TEXT NOT NULL REFERENCES houses(id) ON DELETE CASCADE,
			event_type TEXT NOT NULL,
			actor_id TEXT REFERENCES users(id),
			resource_type TEXT NOT NULL,
			resource_id TEXT NOT NULL,
			payload TEXT NOT NULL DEFAULT '{}',
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`, houseEventCursor),
		`CREATE INDEX IF NOT EXISTS idx_house_events_house_cursor ON house_events(house_id, cursor)`,
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
