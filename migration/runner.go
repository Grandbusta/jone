package migration

import (
	"fmt"
	"slices"

	"github.com/Grandbusta/jone/config"
	"github.com/Grandbusta/jone/schema"
)

// RunOptions holds optional flags for migration commands.
type RunOptions struct {
	All  bool     // For rollback --all (rollback all batches)
	Args []string // Positional arguments
}

// RunParams holds all parameters needed to run migrations.
type RunParams struct {
	Config        *config.Config
	Registrations []Registration
	Schema        *schema.Schema
	Options       RunOptions
}

// RunLatest executes pending Up migrations in order using the provided schema.
// Each migration is wrapped in a transaction.
func RunLatest(p RunParams) error {
	// Create tracker
	d := p.Schema.Dialect()
	tracker := NewTracker(p.Schema.DB(), d, p.Config.Migrations.TableName)

	// Ensure tracking table exists
	if err := tracker.EnsureTable(); err != nil {
		return fmt.Errorf("ensuring migrations table: %w", err)
	}

	// Get applied migrations
	applied, err := tracker.GetApplied()
	if err != nil {
		return fmt.Errorf("getting applied migrations: %w", err)
	}
	appliedSet := make(map[string]bool)
	for _, name := range applied {
		appliedSet[name] = true
	}

	// Filter to pending
	var pending []Registration
	for _, reg := range p.Registrations {
		if !appliedSet[reg.Name] {
			pending = append(pending, reg)
		}
	}

	if len(pending) == 0 {
		fmt.Println("No pending migrations")
		return nil
	}

	// Get next batch number
	lastBatch, err := tracker.GetLastBatch()
	if err != nil {
		return fmt.Errorf("getting last batch: %w", err)
	}
	batch := lastBatch + 1

	fmt.Printf("Running %d migration(s) in batch %d...\n", len(pending), batch)

	// Run each pending migration in a transaction
	for _, reg := range pending {
		if err := runMigration(p, tracker, reg, batch); err != nil {
			return err
		}
	}

	fmt.Println("All migrations completed successfully")
	return nil
}

// RunUp runs the next pending migration or a specific one if Args[0] provided.
// Each migration is wrapped in a transaction.
func RunUp(p RunParams) error {
	d := p.Schema.Dialect()
	tracker := NewTracker(p.Schema.DB(), d, p.Config.Migrations.TableName)

	// Ensure tracking table exists
	if err := tracker.EnsureTable(); err != nil {
		return fmt.Errorf("ensuring migrations table: %w", err)
	}

	// Get applied migrations
	applied, err := tracker.GetApplied()
	if err != nil {
		return fmt.Errorf("getting applied migrations: %w", err)
	}
	appliedSet := make(map[string]bool)
	for _, name := range applied {
		appliedSet[name] = true
	}

	// Build map of registrations for lookup
	regMap := make(map[string]Registration)
	for _, reg := range p.Registrations {
		regMap[reg.Name] = reg
	}

	// Get next batch number
	lastBatch, err := tracker.GetLastBatch()
	if err != nil {
		return fmt.Errorf("getting last batch: %w", err)
	}
	batch := lastBatch + 1

	// Determine which migration to run
	var targetReg Registration
	if len(p.Options.Args) > 0 {
		// Run specific migration by name
		targetName := p.Options.Args[0]
		reg, ok := regMap[targetName]
		if !ok {
			return fmt.Errorf("migration %s not found in registry", targetName)
		}
		if appliedSet[targetName] {
			fmt.Printf("Migration %s already applied\n", targetName)
			return nil
		}
		targetReg = reg
	} else {
		// Run the next pending migration
		var pending []Registration
		for _, reg := range p.Registrations {
			if !appliedSet[reg.Name] {
				pending = append(pending, reg)
			}
		}
		if len(pending) == 0 {
			fmt.Println("No pending migrations")
			return nil
		}
		targetReg = pending[0]
	}

	// Run the migration
	if err := runMigration(p, tracker, targetReg, batch); err != nil {
		return err
	}

	fmt.Println("Migration completed successfully")
	return nil
}

// runMigration runs a single migration in a transaction.
func runMigration(p RunParams, tracker *Tracker, reg Registration, batch int) error {
	tx, err := p.Schema.BeginTx()
	if err != nil {
		return fmt.Errorf("starting transaction: %w", err)
	}

	txSchema := p.Schema.WithTx(tx)
	reg.Up(txSchema)

	if err := tracker.RecordMigrationTx(tx, reg.Name, batch); err != nil {
		tx.Rollback()
		return fmt.Errorf("recording migration %s: %w", reg.Name, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing migration %s: %w", reg.Name, err)
	}

	fmt.Printf("Migrated: %s\n", reg.Name)
	return nil
}

// RunDown rolls back the last single migration that was applied.
// Each migration is wrapped in a transaction.
func RunDown(p RunParams) error {
	d := p.Schema.Dialect()
	tracker := NewTracker(p.Schema.DB(), d, p.Config.Migrations.TableName)

	applied, err := tracker.GetApplied()
	if err != nil {
		return fmt.Errorf("getting applied migrations: %w", err)
	}

	if len(applied) == 0 {
		fmt.Println("No migrations to rollback")
		return nil
	}

	// Build map of registrations for lookup
	regMap := make(map[string]Registration)
	for _, reg := range p.Registrations {
		regMap[reg.Name] = reg
	}

	// Determine which migration to rollback
	var targetMigration string
	if len(p.Options.Args) > 0 {
		// Rollback specific migration by name
		targetMigration = p.Options.Args[0]
		// Verify it's in the applied list
		if !slices.Contains(applied, targetMigration) {
			return fmt.Errorf("migration %s not found in applied migrations", targetMigration)
		}
	} else {
		// Rollback the last applied migration
		targetMigration = applied[len(applied)-1]
	}

	// Rollback the migration
	if err := rollbackMigration(p, tracker, regMap, targetMigration); err != nil {
		return err
	}

	fmt.Println("Rollback completed successfully")
	return nil
}

// RunRollback rolls back the last batch of migrations (or all if Options.All is true).
// Each migration is wrapped in a transaction.
func RunRollback(p RunParams) error {
	d := p.Schema.Dialect()
	tracker := NewTracker(p.Schema.DB(), d, p.Config.Migrations.TableName)

	// Build map of registrations for lookup
	regMap := make(map[string]Registration)
	for _, reg := range p.Registrations {
		regMap[reg.Name] = reg
	}

	if p.Options.All {
		// Rollback all migrations
		return rollbackAll(p, tracker, regMap)
	}

	// Rollback last batch only
	return rollbackLastBatch(p, tracker, regMap)
}

func rollbackLastBatch(p RunParams, tracker *Tracker, regMap map[string]Registration) error {
	lastBatch, err := tracker.GetLastBatch()
	if err != nil {
		return fmt.Errorf("getting last batch: %w", err)
	}

	if lastBatch == 0 {
		fmt.Println("Nothing to rollback.")
		return nil
	}

	batchMigrations, err := tracker.GetBatchMigrations(lastBatch)
	if err != nil {
		return fmt.Errorf("getting batch migrations: %w", err)
	}

	if len(batchMigrations) == 0 {
		fmt.Println("Nothing to rollback.")
		return nil
	}

	fmt.Printf("Rolling back %d migration(s) from batch %d...\n", len(batchMigrations), lastBatch)

	for _, name := range batchMigrations {
		if err := rollbackMigration(p, tracker, regMap, name); err != nil {
			return err
		}
	}

	fmt.Println("Rollback completed successfully")
	return nil
}

func rollbackAll(p RunParams, tracker *Tracker, regMap map[string]Registration) error {
	applied, err := tracker.GetApplied()
	if err != nil {
		return fmt.Errorf("getting applied migrations: %w", err)
	}

	if len(applied) == 0 {
		fmt.Println("Nothing to rollback.")
		return nil
	}

	fmt.Printf("Rolling back all %d migration(s)...\n", len(applied))

	// Roll back in reverse order
	for i := len(applied) - 1; i >= 0; i-- {
		if err := rollbackMigration(p, tracker, regMap, applied[i]); err != nil {
			return err
		}
	}

	fmt.Println("Rollback completed successfully")
	return nil
}

func rollbackMigration(p RunParams, tracker *Tracker, regMap map[string]Registration, name string) error {
	reg, ok := regMap[name]
	if !ok {
		return fmt.Errorf("migration %s not found in registry", name)
	}

	tx, err := p.Schema.BeginTx()
	if err != nil {
		return fmt.Errorf("starting transaction: %w", err)
	}

	txSchema := p.Schema.WithTx(tx)
	reg.Down(txSchema)

	if err := tracker.RemoveMigrationTx(tx, name); err != nil {
		tx.Rollback()
		return fmt.Errorf("removing migration record %s: %w", name, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing rollback %s: %w", name, err)
	}

	fmt.Printf("Rolled back: %s\n", name)
	return nil
}
