package database

import "fmt"

func householdStatements(driver string) []string {
	chatCursor := `INTEGER PRIMARY KEY AUTOINCREMENT`
	if driver == "postgres" {
		chatCursor = `BIGSERIAL PRIMARY KEY`
	}
	return []string{
		`CREATE TABLE IF NOT EXISTS grocery_items (
			id TEXT PRIMARY KEY,
			house_id TEXT NOT NULL REFERENCES houses(id) ON DELETE CASCADE,
			creator_id TEXT NOT NULL REFERENCES users(id),
			name TEXT NOT NULL CHECK(length(trim(name)) BETWEEN 1 AND 160),
			quantity TEXT NOT NULL DEFAULT '' CHECK(length(quantity) <= 40),
			unit TEXT NOT NULL DEFAULT '' CHECK(length(unit) <= 40),
			note TEXT NOT NULL DEFAULT '' CHECK(length(note) <= 1000),
			assignee_id TEXT REFERENCES users(id),
			checked BOOLEAN NOT NULL DEFAULT FALSE,
			checked_by TEXT REFERENCES users(id),
			checked_at TIMESTAMP,
			position INTEGER NOT NULL DEFAULT 0,
			version BIGINT NOT NULL DEFAULT 1,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_grocery_items_house_order ON grocery_items(house_id,position,created_at,id)`,
		`CREATE TABLE IF NOT EXISTS chores (
			id TEXT PRIMARY KEY,
			house_id TEXT NOT NULL REFERENCES houses(id) ON DELETE CASCADE,
			creator_id TEXT NOT NULL REFERENCES users(id),
			title TEXT NOT NULL CHECK(length(trim(title)) BETWEEN 1 AND 160),
			description TEXT NOT NULL DEFAULT '' CHECK(length(description) <= 4000),
			assignee_id TEXT REFERENCES users(id),
			timezone TEXT NOT NULL,
			due_local TEXT NOT NULL,
			rrule TEXT NOT NULL,
			exdates TEXT NOT NULL DEFAULT '[]',
			enabled BOOLEAN NOT NULL DEFAULT TRUE,
			version BIGINT NOT NULL DEFAULT 1,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_chores_house ON chores(house_id,created_at,id)`,
		`CREATE TABLE IF NOT EXISTS chore_completions (
			id TEXT PRIMARY KEY,
			chore_id TEXT NOT NULL REFERENCES chores(id) ON DELETE CASCADE,
			house_id TEXT NOT NULL REFERENCES houses(id) ON DELETE CASCADE,
			occurrence_at TIMESTAMP NOT NULL,
			completed_by TEXT NOT NULL REFERENCES users(id),
			completed_at TIMESTAMP NOT NULL,
			UNIQUE(chore_id,occurrence_at)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_chore_completions_house ON chore_completions(house_id,completed_at DESC)`,
		`CREATE TABLE IF NOT EXISTS calendar_events (
			id TEXT PRIMARY KEY,
			house_id TEXT NOT NULL REFERENCES houses(id) ON DELETE CASCADE,
			creator_id TEXT NOT NULL REFERENCES users(id),
			title TEXT NOT NULL CHECK(length(trim(title)) BETWEEN 1 AND 160),
			description TEXT NOT NULL DEFAULT '' CHECK(length(description) <= 4000),
			timezone TEXT NOT NULL,
			start_local TEXT NOT NULL,
			end_local TEXT NOT NULL,
			all_day BOOLEAN NOT NULL DEFAULT FALSE,
			rrule TEXT NOT NULL,
			exdates TEXT NOT NULL DEFAULT '[]',
			version BIGINT NOT NULL DEFAULT 1,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_calendar_events_house ON calendar_events(house_id,created_at,id)`,
		fmt.Sprintf(`CREATE TABLE IF NOT EXISTS chat_messages (
			cursor %s,
			id TEXT NOT NULL UNIQUE,
			house_id TEXT NOT NULL REFERENCES houses(id) ON DELETE CASCADE,
			author_id TEXT NOT NULL REFERENCES users(id),
			body TEXT,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			deleted_at TIMESTAMP,
			redacted_at TIMESTAMP
		)`, chatCursor),
		`CREATE INDEX IF NOT EXISTS idx_chat_messages_house_cursor ON chat_messages(house_id,cursor DESC)`,
		`CREATE TABLE IF NOT EXISTS chat_retention_jobs (
			id TEXT PRIMARY KEY,
			available_at TIMESTAMP NOT NULL,
			lease_owner TEXT,
			lease_expires_at TIMESTAMP,
			lease_generation BIGINT NOT NULL DEFAULT 0,
			last_completed_at TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`INSERT INTO chat_retention_jobs(id,available_at) VALUES('chat-body-retention','1970-01-01 00:00:00') ON CONFLICT(id) DO NOTHING`,
	}
}
