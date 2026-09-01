package database

import (
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

func Connect(databaseURL string) (*sqlx.DB, error) {
	driver, dsn := parseURL(databaseURL)

	db, err := sqlx.Connect(driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("connect to database: %w", err)
	}

	if driver == "sqlite3" {
		// SQLite pragmas are connection-local, so keep a single connection to
		// guarantee that foreign-key enforcement cannot be bypassed by the pool.
		db.SetMaxOpenConns(1)
		db.SetMaxIdleConns(1)
		if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
			db.Close()
			return nil, fmt.Errorf("enable sqlite foreign keys: %w", err)
		}
		if _, err := db.Exec("PRAGMA busy_timeout = 5000"); err != nil {
			db.Close()
			return nil, fmt.Errorf("set sqlite busy timeout: %w", err)
		}
	} else {
		db.SetMaxOpenConns(25)
		db.SetMaxIdleConns(5)
		db.SetConnMaxLifetime(30 * time.Minute)
		db.SetConnMaxIdleTime(5 * time.Minute)
	}

	return db, nil
}

func RunMigrations(db *sqlx.DB) error {
	tx, err := db.Beginx()
	if err != nil {
		return fmt.Errorf("begin migrations: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY,
		applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`); err != nil {
		return fmt.Errorf("create migration ledger: %w", err)
	}

	for i, migration := range migrations() {
		version := i + 1
		var applied int
		if err := tx.Get(&applied, "SELECT COUNT(*) FROM schema_migrations WHERE version = $1", version); err != nil {
			return fmt.Errorf("check migration %d: %w", version, err)
		}
		if applied != 0 {
			continue
		}
		if _, err := tx.Exec(migration); err != nil {
			return fmt.Errorf("migration %d: %w", version, err)
		}
		if _, err := tx.Exec("INSERT INTO schema_migrations (version) VALUES ($1)", version); err != nil {
			return fmt.Errorf("record migration %d: %w", version, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit migrations: %w", err)
	}
	log.Println("Migrations complete")
	return nil
}

func parseURL(s string) (driver, dsn string) {
	if strings.HasPrefix(s, "sqlite://") {
		path := strings.TrimPrefix(s, "sqlite://")
		return "sqlite3", path
	}

	if strings.HasPrefix(s, "postgres://") || strings.HasPrefix(s, "postgresql://") {
		u, err := url.Parse(s)
		if err == nil {
			q := u.Query()
			if q.Get("sslmode") == "" {
				q.Set("sslmode", "require")
				u.RawQuery = q.Encode()
			}
			return "postgres", u.String()
		}
		return "postgres", s
	}

	if strings.HasPrefix(s, "file:") || strings.HasSuffix(s, ".db") {
		return "sqlite3", s
	}

	return "sqlite3", s
}
