package database

func migrations() []string {
	return []string{
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
}
