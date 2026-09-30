package database

func notificationStatements() []string {
	return []string{
		`CREATE TABLE IF NOT EXISTS notification_preferences (
			house_id TEXT NOT NULL REFERENCES houses(id) ON DELETE CASCADE,
			user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			expense_created_enabled BOOLEAN NOT NULL DEFAULT TRUE,
			reminder_enabled BOOLEAN NOT NULL DEFAULT TRUE,
			cadence TEXT NOT NULL DEFAULT 'immediate' CHECK(cadence IN ('immediate','daily_digest')),
			timezone TEXT NOT NULL DEFAULT 'UTC',
			quiet_start_minutes INTEGER CHECK(quiet_start_minutes BETWEEN 0 AND 1439),
			quiet_end_minutes INTEGER CHECK(quiet_end_minutes BETWEEN 0 AND 1439),
			digest_minutes INTEGER NOT NULL DEFAULT 480 CHECK(digest_minutes BETWEEN 0 AND 1439),
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY(house_id, user_id)
		)`,
		`CREATE TABLE IF NOT EXISTS notification_subscriptions (
			id TEXT PRIMARY KEY,
			house_id TEXT NOT NULL REFERENCES houses(id) ON DELETE CASCADE,
			user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			platform TEXT NOT NULL CHECK(platform IN ('web_push','android_fcm')),
			identity_hash TEXT NOT NULL,
			key_id TEXT NOT NULL,
			ciphertext TEXT NOT NULL,
			device_label TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			last_seen_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			revoked_at TIMESTAMP,
			UNIQUE(house_id, user_id, platform, identity_hash)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_notification_subscriptions_owner ON notification_subscriptions(house_id,user_id,revoked_at)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_notification_subscriptions_active_identity ON notification_subscriptions(platform,identity_hash) WHERE revoked_at IS NULL`,
		`CREATE TABLE IF NOT EXISTS scheduled_house_events (
			id TEXT PRIMARY KEY,
			house_id TEXT NOT NULL REFERENCES houses(id) ON DELETE CASCADE,
			creator_id TEXT NOT NULL REFERENCES users(id),
			title TEXT NOT NULL CHECK(length(trim(title)) BETWEEN 1 AND 120),
			timezone TEXT NOT NULL,
			dtstart_local TEXT NOT NULL,
			rrule TEXT NOT NULL,
			exdates TEXT NOT NULL DEFAULT '[]',
			next_occurrence_at TIMESTAMP,
			enabled BOOLEAN NOT NULL DEFAULT TRUE,
			lease_owner TEXT,
			lease_expires_at TIMESTAMP,
			lease_generation BIGINT NOT NULL DEFAULT 0,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_scheduled_events_due ON scheduled_house_events(enabled,next_occurrence_at,lease_expires_at)`,
		`CREATE TABLE IF NOT EXISTS notification_deliveries (
			id TEXT PRIMARY KEY,
			house_id TEXT NOT NULL REFERENCES houses(id) ON DELETE CASCADE,
			user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			category TEXT NOT NULL CHECK(category IN ('expense_created','reminder')),
			resource_type TEXT NOT NULL,
			resource_id TEXT NOT NULL,
			occurrence_at TIMESTAMP NOT NULL,
			cadence TEXT NOT NULL CHECK(cadence IN ('immediate','daily_digest')),
			available_at TIMESTAMP NOT NULL,
			dispatched_at TIMESTAMP,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(house_id,user_id,category,resource_type,resource_id,occurrence_at)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_notification_deliveries_due ON notification_deliveries(dispatched_at,available_at)`,
	}
}
