package main

import (
	"efournierrobert/cake-backend/internal/repository"
	"fmt"
	"log"
)

func main() {
	log.Println("Starting cake...")

	db, err := repository.NewDbConnection()
	if err != nil {
		log.Fatalf("error while opening connection to database: %s", fmt.Errorf("%w", err))
	}
	defer db.Close()
	// Create HTTP server
	// Initialize handlers
	// Start HTTP server
}
