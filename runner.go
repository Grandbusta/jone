package jone

import "fmt"

// RunUp executes all Up migrations in order.
func RunUp(registrations []Registration) {
	for _, reg := range registrations {
		fmt.Printf("Running migration: %s (up)\n", reg.Name)
		schema := Schema{}
		reg.Up(schema)
		fmt.Printf("Completed migration: %s\n", reg.Name)
	}
	fmt.Println("All migrations completed successfully")
}

// RunDown executes all Down migrations in reverse order.
func RunDown(registrations []Registration) {
	// Run in reverse order
	for i := len(registrations) - 1; i >= 0; i-- {
		reg := registrations[i]
		fmt.Printf("Rolling back migration: %s (down)\n", reg.Name)
		schema := Schema{}
		reg.Down(schema)
		fmt.Printf("Completed rollback: %s\n", reg.Name)
	}
	fmt.Println("All rollbacks completed successfully")
}
