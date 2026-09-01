package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

const migrationAdvisoryLockKey int64 = 5479457351723385152

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
	ctx := context.Background()
	switch db.DriverName() {
	case "sqlite3":
		return runSQLiteMigrations(ctx, db)
	case "postgres":
		return runPostgresMigrations(ctx, db)
	default:
		return runPostgresMigrations(ctx, db)
	}
}

func runSQLiteMigrations(ctx context.Context, db *sqlx.DB) (err error) {
	conn, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("acquire sqlite migration connection: %w", err)
	}
	defer conn.Close()

	if _, err := conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return fmt.Errorf("begin sqlite migration lock: %w", err)
	}
	committed := false
	defer func() {
		if committed {
			return
		}
		if rollbackErr := rollbackMigrationContext(ctx, conn); rollbackErr != nil && err == nil {
			err = rollbackErr
		}
	}()

	if err := applyMigrations(ctx, conn, db.DriverName()); err != nil {
		return err
	}
	if _, err := conn.ExecContext(ctx, "COMMIT"); err != nil {
		return fmt.Errorf("commit sqlite migrations: %w", err)
	}
	committed = true
	log.Println("Migrations complete")
	return nil
}

func runPostgresMigrations(ctx context.Context, db *sqlx.DB) error {
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin migrations: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock($1)", migrationAdvisoryLockKey); err != nil {
		return fmt.Errorf("acquire migration lock: %w", err)
	}
	if err := applyMigrations(ctx, tx, db.DriverName()); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit migrations: %w", err)
	}
	log.Println("Migrations complete")
	return nil
}

func applyMigrations(ctx context.Context, exec migrationRunner, driver string) error {
	if _, err := exec.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY,
		name TEXT NOT NULL,
		applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`); err != nil {
		return fmt.Errorf("create migration ledger: %w", err)
	}
	if err := ensureMigrationLedgerCompatibility(ctx, exec, driver); err != nil {
		return err
	}

	for _, migration := range migrations(driver) {
		var applied int
		if err := exec.QueryRowContext(ctx, "SELECT COUNT(*) FROM schema_migrations WHERE version = $1", migration.version).Scan(&applied); err != nil {
			return fmt.Errorf("check migration %d: %w", migration.version, err)
		}
		if applied != 0 {
			continue
		}
		if err := migration.up(ctx, exec); err != nil {
			return fmt.Errorf("migration %d (%s): %w", migration.version, migration.name, err)
		}
		if _, err := exec.ExecContext(ctx, "INSERT INTO schema_migrations (version, name) VALUES ($1, $2)", migration.version, migration.name); err != nil {
			return fmt.Errorf("record migration %d: %w", migration.version, err)
		}
	}
	return nil
}

func rollbackMigrationContext(ctx context.Context, conn interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}) error {
	if _, err := conn.ExecContext(ctx, "ROLLBACK"); err != nil {
		return fmt.Errorf("rollback sqlite migrations: %w", err)
	}
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
