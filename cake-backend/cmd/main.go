package main

import (
	"efournierrobert/cake-backend/internal/repository"
	"fmt"
	"log"
	"net/http"
)

func main() {
	log.Println("Starting cake...")

	db, err := repository.NewDbConnection()
	if err != nil {
		log.Fatalf("error while opening connection to database: %s", fmt.Errorf("%w", err))
	}
	defer db.Close()

	mux := http.NewServeMux()
	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	log.Println("Cake server started and listening on port 8080")
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("error while http server was running: %s", fmt.Errorf("%w", err))
	}
}
