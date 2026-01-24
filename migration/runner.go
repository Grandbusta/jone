package migration

import (
	"fmt"
	"slices"

	"github.com/Grandbusta/jone/config"
	"github.com/Grandbusta/jone/internal/term"
	"github.com/Grandbusta/jone/schema"
)

// RunOptions holds optional flags for migration commands.
type RunOptions struct {
	All    bool     // For rollback --all (rollback all batches)
	DryRun bool     // Show SQL without executing
	Args   []string // Positional arguments
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
	// Dry-run mode: just show what would be executed
	if p.Options.DryRun {
		return runLatestDryRun(p)
	}

	// Create tracker
	d := p.Schema.Dialect()
	tracker := NewTracker(p.Schema.DB(), d, p.Config.Migrations.TableName)

	// Ensure tracking table exists
	if err := tracker.EnsureTable(); err != nil {
		return err // Error already descriptive from tracker
	}

	// Get applied migrations
	applied, err := tracker.GetApplied()
	if err != nil {
		return err // Error already descriptive from tracker
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
		fmt.Println(term.YellowText("No pending migrations"))
		return nil
	}

	// Get next batch number
	lastBatch, err := tracker.GetLastBatch()
	if err != nil {
		return err // Error already descriptive from tracker
	}
	batch := lastBatch + 1

	fmt.Println(term.CyanText(fmt.Sprintf("Running %d migration(s) in batch %d...", len(pending), batch)))

	// Run each pending migration in a transaction
	for _, reg := range pending {
		if err := runMigration(p, tracker, reg, batch); err != nil {
			return err
		}
	}

	fmt.Println(term.GreenText("✓ All migrations completed successfully"))
	return nil
}

// runLatestDryRun shows what migrations would be run without executing.
func runLatestDryRun(p RunParams) error {
	fmt.Println(term.YellowText("[DRY RUN]") + " Would run the following migrations:")
	fmt.Println()

	for _, reg := range p.Registrations {
		fmt.Printf("Migration: %s\n", term.GreenText(reg.Name))
		fmt.Println("SQL:")
		reg.Up(p.Schema) // Schema has no execer, so it prints SQL
		fmt.Println()
	}

	fmt.Printf("Total: %d migration(s) would be applied\n", len(p.Registrations))
	return nil
}

// RunList displays all migrations with their status (applied/pending).
func RunList(p RunParams) error {
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

	// Count stats
	appliedCount := 0
	pendingCount := 0

	fmt.Println("\nMigrations:")
	fmt.Println("───────────────────────────────────────────────────────")

	for _, reg := range p.Registrations {
		if appliedSet[reg.Name] {
			fmt.Printf("  %s  %s\n", term.GreenText("✓"), reg.Name)
			appliedCount++
		} else {
			fmt.Printf("  %s  %s\n", term.YellowText("○"), reg.Name)
			pendingCount++
		}
	}

	fmt.Println("───────────────────────────────────────────────────────")
	fmt.Printf("Total: %s, %s\n\n", term.GreenText(fmt.Sprintf("%d applied", appliedCount)), term.YellowText(fmt.Sprintf("%d pending", pendingCount)))

	return nil
}

// RunUp runs the next pending migration or a specific one if Args[0] provided.
// Each migration is wrapped in a transaction.
func RunUp(p RunParams) error {
	// Dry-run mode
	if p.Options.DryRun {
		return runUpDryRun(p)
	}

	d := p.Schema.Dialect()
	tracker := NewTracker(p.Schema.DB(), d, p.Config.Migrations.TableName)

	// Ensure tracking table exists
	if err := tracker.EnsureTable(); err != nil {
		return err
	}

	// Get applied migrations
	applied, err := tracker.GetApplied()
	if err != nil {
		return err
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
		return err
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
			fmt.Println(term.YellowText(fmt.Sprintf("Migration %s already applied", targetName)))
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
			fmt.Println(term.YellowText("No pending migrations"))
			return nil
		}
		targetReg = pending[0]
	}

	// Run the migration
	if err := runMigration(p, tracker, targetReg, batch); err != nil {
		return err
	}

	fmt.Println(term.GreenText("✓ Migration completed successfully"))
	return nil
}

// runMigration runs a single migration in a transaction.
func runMigration(p RunParams, tracker *Tracker, reg Registration, batch int) error {
	tx, err := p.Schema.BeginTx()
	if err != nil {
		return fmt.Errorf("failed to start transaction for '%s': %w", reg.Name, err)
	}

	txSchema := p.Schema.WithTx(tx)
	reg.Up(txSchema)

	if err := tracker.RecordMigrationTx(tx, reg.Name, batch); err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to record migration '%s': %w", reg.Name, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit migration '%s': %w", reg.Name, err)
	}

	fmt.Println(term.GreenText(fmt.Sprintf("  ✓ Migrated: %s", reg.Name)))
	return nil
}

// RunDown rolls back the last single migration that was applied.
// Each migration is wrapped in a transaction.
func RunDown(p RunParams) error {
	// Dry-run mode
	if p.Options.DryRun {
		return runDownDryRun(p)
	}

	d := p.Schema.Dialect()
	tracker := NewTracker(p.Schema.DB(), d, p.Config.Migrations.TableName)

	applied, err := tracker.GetApplied()
	if err != nil {
		return err
	}

	if len(applied) == 0 {
		fmt.Println(term.YellowText("No migrations to rollback"))
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

	fmt.Println(term.GreenText("✓ Rollback completed successfully"))
	return nil
}

// RunRollback rolls back the last batch of migrations (or all if Options.All is true).
// Each migration is wrapped in a transaction.
func RunRollback(p RunParams) error {
	// Dry-run mode
	if p.Options.DryRun {
		return runRollbackDryRun(p)
	}

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
		fmt.Println(term.YellowText("Nothing to rollback."))
		return nil
	}

	batchMigrations, err := tracker.GetBatchMigrations(lastBatch)
	if err != nil {
		return fmt.Errorf("getting batch migrations: %w", err)
	}

	if len(batchMigrations) == 0 {
		fmt.Println(term.YellowText("Nothing to rollback."))
		return nil
	}

	fmt.Println(term.CyanText(fmt.Sprintf("Rolling back %d migration(s) from batch %d...", len(batchMigrations), lastBatch)))

	for _, name := range batchMigrations {
		if err := rollbackMigration(p, tracker, regMap, name); err != nil {
			return err
		}
	}

	fmt.Println(term.GreenText("✓ Rollback completed successfully"))
	return nil
}

func rollbackAll(p RunParams, tracker *Tracker, regMap map[string]Registration) error {
	applied, err := tracker.GetApplied()
	if err != nil {
		return fmt.Errorf("getting applied migrations: %w", err)
	}

	if len(applied) == 0 {
		fmt.Println(term.YellowText("Nothing to rollback."))
		return nil
	}

	fmt.Println(term.CyanText(fmt.Sprintf("Rolling back all %d migration(s)...", len(applied))))

	// Roll back in reverse order
	for i := len(applied) - 1; i >= 0; i-- {
		if err := rollbackMigration(p, tracker, regMap, applied[i]); err != nil {
			return err
		}
	}

	fmt.Println(term.GreenText("✓ Rollback completed successfully"))
	return nil
}

func rollbackMigration(p RunParams, tracker *Tracker, regMap map[string]Registration, name string) error {
	reg, ok := regMap[name]
	if !ok {
		return fmt.Errorf("migration '%s' not found in registry. Was it deleted or renamed?", name)
	}

	tx, err := p.Schema.BeginTx()
	if err != nil {
		return fmt.Errorf("failed to start transaction for rollback '%s': %w", name, err)
	}

	txSchema := p.Schema.WithTx(tx)
	reg.Down(txSchema)

	if err := tracker.RemoveMigrationTx(tx, name); err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to remove migration record '%s': %w", name, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit rollback '%s': %w", name, err)
	}

	fmt.Println(term.GreenText(fmt.Sprintf("  ✓ Rolled back: %s", name)))
	return nil
}

// runUpDryRun shows what migration would be run without executing.
func runUpDryRun(p RunParams) error {
	// Build map of registrations
	regMap := make(map[string]Registration)
	for _, reg := range p.Registrations {
		regMap[reg.Name] = reg
	}

	var targetReg Registration
	if len(p.Options.Args) > 0 {
		// Specific migration
		targetName := p.Options.Args[0]
		reg, ok := regMap[targetName]
		if !ok {
			return fmt.Errorf("migration %s not found in registry", targetName)
		}
		targetReg = reg
	} else {
		// First pending (just use first registered for dry-run)
		if len(p.Registrations) == 0 {
			fmt.Println(term.YellowText("No migrations registered"))
			return nil
		}
		targetReg = p.Registrations[0]
	}

	fmt.Println(term.YellowText("[DRY RUN]") + " Would run migration:")
	fmt.Println()
	fmt.Printf("Migration: %s\n", term.GreenText(targetReg.Name))
	fmt.Println("SQL:")
	targetReg.Up(p.Schema)
	fmt.Println()
	return nil
}

// runDownDryRun shows what migration would be rolled back without executing.
func runDownDryRun(p RunParams) error {
	// Build map of registrations
	regMap := make(map[string]Registration)
	for _, reg := range p.Registrations {
		regMap[reg.Name] = reg
	}

	var targetName string
	if len(p.Options.Args) > 0 {
		targetName = p.Options.Args[0]
	} else if len(p.Registrations) > 0 {
		// In dry-run we can't check DB, so use last registered
		targetName = p.Registrations[len(p.Registrations)-1].Name
	} else {
		fmt.Println(term.YellowText("No migrations registered"))
		return nil
	}

	reg, ok := regMap[targetName]
	if !ok {
		return fmt.Errorf("migration %s not found in registry", targetName)
	}

	fmt.Println(term.YellowText("[DRY RUN]") + " Would rollback migration:")
	fmt.Println()
	fmt.Printf("Migration: %s\n", term.GreenText(reg.Name))
	fmt.Println("SQL:")
	reg.Down(p.Schema)
	fmt.Println()
	return nil
}

// runRollbackDryRun shows what migrations would be rolled back without executing.
func runRollbackDryRun(p RunParams) error {
	// Build map of registrations
	regMap := make(map[string]Registration)
	for _, reg := range p.Registrations {
		regMap[reg.Name] = reg
	}

	if len(p.Registrations) == 0 {
		fmt.Println(term.YellowText("No migrations registered"))
		return nil
	}

	fmt.Println(term.YellowText("[DRY RUN]") + " Would rollback migrations:")
	fmt.Println()

	if p.Options.All {
		// All migrations in reverse order
		for i := len(p.Registrations) - 1; i >= 0; i-- {
			reg := p.Registrations[i]
			fmt.Printf("Migration: %s\n", term.GreenText(reg.Name))
			fmt.Println("SQL:")
			reg.Down(p.Schema)
			fmt.Println()
		}
		fmt.Printf("Total: %d migration(s) would be rolled back\n", len(p.Registrations))
	} else {
		// Just the last one
		reg := p.Registrations[len(p.Registrations)-1]
		fmt.Printf("Migration: %s\n", term.GreenText(reg.Name))
		fmt.Println("SQL:")
		reg.Down(p.Schema)
		fmt.Println()
	}

	return nil
}
