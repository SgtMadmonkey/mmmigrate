package mmmigrate

import (
	"context"
	"database/sql"
	"fmt"
	"os"
)

type Migration struct {
	Name string //Inferred from file name
	Up   func(tx *sql.Tx) error
	Down func(tx *sql.Tx) error
}

type Migrator struct {
	db            *sql.DB
	migrationsDir string
}

/*
DefaultMigrationsDir is the directory, relative to the consuming project's root, where migration files are generated and read from unless overridden.
*/
const DefaultMigrationsDir = "internal/migrations/"

func New(db *sql.DB) (*Migrator, error) {
	if db == nil {
		return nil, fmt.Errorf("migrate: db is nil")
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("migrate: connecting to db: %w", err)
	}

	info, err := os.Stat(DefaultMigrationsDir)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("migrate: migrations dir %q does not exist — run `migrate create` first", DefaultMigrationsDir)
	}
	if err != nil {
		return nil, fmt.Errorf("migrate: migrations dir %q: %w", DefaultMigrationsDir, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("migrate: %q is not a directory", DefaultMigrationsDir)
	}

	return &Migrator{db: db, migrationsDir: DefaultMigrationsDir}, nil
}

const tableMigrations = "schema_migration"

func ensureMigrationsTable(ctx context.Context, db *sql.DB) error {
	cmd := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
			name TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`, tableMigrations)
	_, err := db.ExecContext(ctx, cmd)
	return err
}

func isApplied(ctx context.Context, db *sql.DB, name string) (bool, error) {
	var exists bool
	query := fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM %s WHERE name = $1)", tableMigrations)
	err := db.QueryRowContext(ctx, query, name).Scan(&exists)
	return exists, err
}

// Runs command within a SQL transaction. This does not necessarily guarantee complete rollbacks on failure
func runSafely(ctx context.Context, db *sql.DB, m Migration, up bool) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	step := m.Down
	if up {
		step = m.Up
	}

	//Run transaction
	if err := step(tx); err != nil {
		return err
	}

	if up {
		cmd := fmt.Sprintf(`INSERT INTO %s (name) VALUES ($1)`, tableMigrations)
		if _, err := tx.ExecContext(ctx, cmd, m.Name); err != nil {
			return fmt.Errorf("record migration history: %w", err)
		}
	} else {
		cmd := fmt.Sprintf(`DELETE FROM %s WHERE name = $1`, tableMigrations)
		if _, err := tx.ExecContext(ctx, cmd, m.Name); err != nil {
			return fmt.Errorf("remove migration history: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("migration commit: %w", err)
	}
	return nil
}

func (migrator *Migrator) Up(ctx context.Context, migrations []Migration) error {
	//Create migrations table if doesn't exist
	if err := ensureMigrationsTable(ctx, migrator.db); err != nil {
		return fmt.Errorf("creating migrations table: %w", err)
	}

	for _, m := range migrations {
		applied, err := isApplied(ctx, migrator.db, m.Name)
		if err != nil {
			return fmt.Errorf("checking migration %s: %w", m.Name, err)
		}
		if applied {
			//Already applied this migration - skip
			fmt.Printf("migration %s already applied, skipping...\n", m.Name)
			continue
		}

		if err := runSafely(ctx, migrator.db, m, true); err != nil {
			return fmt.Errorf("migration %s failed: %w", m.Name, err)
		}
		fmt.Printf("migration %s applied!\n", m.Name)
	}
	return nil
}

// Rolls back the single most recently applied migration
func (migrator *Migrator) Down(ctx context.Context, migrations []Migration) error {
	var name string
	query := fmt.Sprintf("SELECT name FROM %s ORDER BY name DESC LIMIT 1", tableMigrations)
	err := migrator.db.QueryRowContext(ctx, query).Scan(&name)
	if err == sql.ErrNoRows {
		fmt.Println("no migrations to roll back")
		return nil
	}
	if err != nil {
		return fmt.Errorf("find last migration: %w", err)
	}

	var target *Migration
	for i := range migrations {
		if migrations[i].Name == name {
			target = &migrations[i]
			break
		}
	}
	if target == nil {
		return fmt.Errorf("migration %s not found in registry", name)
	}

	if err := runSafely(ctx, migrator.db, *target, false); err != nil {
		return fmt.Errorf("rollback %s failed: %w", name, err)
	}
	fmt.Printf("migration %s rolled back\n", name)
	return nil
}
