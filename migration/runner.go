package migration

import (
	"fmt"

	"github.com/Grandbusta/jone/config"
	"github.com/Grandbusta/jone/schema"
)

// RunUp executes all Up migrations in order using the provided config.
func RunLatest(cfg *config.Config, registrations []Registration) error {
	s := schema.New(cfg)

	// TODO: Connect to database using cfg.Connection
	// For now, running in dry-run mode

	for _, reg := range registrations {
		fmt.Printf("Running migration: %s (up)\n", reg.Name)
		reg.Up(s)
		fmt.Printf("Completed migration: %s\n", reg.Name)
	}
	fmt.Println("All migrations completed successfully")
	return nil
}

// RunDown executes all Down migrations in reverse order using the provided config.
func RunDown(cfg *config.Config, registrations []Registration) error {
	s := schema.New(cfg)

	// TODO: Connect to database using cfg.Connection
	// For now, running in dry-run mode

	// Run in reverse order
	for i := len(registrations) - 1; i >= 0; i-- {
		reg := registrations[i]
		fmt.Printf("Rolling back migration: %s (down)\n", reg.Name)
		reg.Down(s)
		fmt.Printf("Completed rollback: %s\n", reg.Name)
	}
	fmt.Println("All rollbacks completed successfully")
	return nil
}
