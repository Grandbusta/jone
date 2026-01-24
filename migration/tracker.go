package migration

import (
	"database/sql"
	"fmt"

	"github.com/Grandbusta/jone/dialect"
)

// Tracker handles migration tracking in the database.
type Tracker struct {
	db        *sql.DB
	dialect   dialect.Dialect
	tableName string
}

// NewTracker creates a new migration tracker.
func NewTracker(db *sql.DB, d dialect.Dialect, tableName string) *Tracker {
	if tableName == "" {
		tableName = "jone_migrations"
	}
	return &Tracker{
		db:        db,
		dialect:   d,
		tableName: tableName,
	}
}

// EnsureTable creates the migrations tracking table if it doesn't exist.
func (t *Tracker) EnsureTable() error {
	sql := t.dialect.CreateMigrationsTableSQL(t.tableName)
	_, err := t.db.Exec(sql)
	if err != nil {
		return fmt.Errorf("failed to create migrations table '%s': %w", t.tableName, err)
	}
	return nil
}

// GetApplied returns the list of applied migration names in order.
func (t *Tracker) GetApplied() ([]string, error) {
	sql := t.dialect.GetAppliedMigrationsSQL(t.tableName)
	rows, err := t.db.Query(sql)
	if err != nil {
		return nil, fmt.Errorf("failed to query migrations from '%s': %w", t.tableName, err)
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scanning migration name: %w", err)
		}
		names = append(names, name)
	}
	return names, rows.Err()
}

// GetLastBatch returns the highest batch number, or 0 if no migrations exist.
func (t *Tracker) GetLastBatch() (int, error) {
	sql := t.dialect.GetLastBatchSQL(t.tableName)
	var batch int
	err := t.db.QueryRow(sql).Scan(&batch)
	if err != nil {
		return 0, fmt.Errorf("failed to get last batch number from '%s': %w", t.tableName, err)
	}
	return batch, nil
}

// GetBatchMigrations returns migration names for a specific batch in reverse order.
func (t *Tracker) GetBatchMigrations(batch int) ([]string, error) {
	sql := t.dialect.GetMigrationsByBatchSQL(t.tableName)
	rows, err := t.db.Query(sql, batch)
	if err != nil {
		return nil, fmt.Errorf("failed to query batch %d migrations: %w", batch, err)
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scanning migration name: %w", err)
		}
		names = append(names, name)
	}
	return names, rows.Err()
}

// RecordMigration inserts a record for a successfully run migration.
func (t *Tracker) RecordMigration(name string, batch int) error {
	sql := t.dialect.InsertMigrationSQL(t.tableName)
	_, err := t.db.Exec(sql, name, batch)
	if err != nil {
		return fmt.Errorf("recording migration %s: %w", name, err)
	}
	return nil
}

// RecordMigrationTx inserts a record using the provided transaction.
func (t *Tracker) RecordMigrationTx(tx *sql.Tx, name string, batch int) error {
	sql := t.dialect.InsertMigrationSQL(t.tableName)
	_, err := tx.Exec(sql, name, batch)
	if err != nil {
		return fmt.Errorf("recording migration %s: %w", name, err)
	}
	return nil
}

// RemoveMigration deletes a migration record.
func (t *Tracker) RemoveMigration(name string) error {
	sql := t.dialect.DeleteMigrationSQL(t.tableName)
	_, err := t.db.Exec(sql, name)
	if err != nil {
		return fmt.Errorf("removing migration %s: %w", name, err)
	}
	return nil
}

// RemoveMigrationTx deletes a migration record using the provided transaction.
func (t *Tracker) RemoveMigrationTx(tx *sql.Tx, name string) error {
	sql := t.dialect.DeleteMigrationSQL(t.tableName)
	_, err := tx.Exec(sql, name)
	if err != nil {
		return fmt.Errorf("removing migration %s: %w", name, err)
	}
	return nil
}
