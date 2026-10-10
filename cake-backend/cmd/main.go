package main

import (
	"crypto/rand"
	"efournierrobert/cake-backend/internal/encryptor"
	providerHandler "efournierrobert/cake-backend/internal/handlers/providers"
	userHandler "efournierrobert/cake-backend/internal/handlers/users"
	"efournierrobert/cake-backend/internal/repository"
	userRepo "efournierrobert/cake-backend/internal/repository/users"
	providerService "efournierrobert/cake-backend/internal/services/providers"
	"efournierrobert/cake-backend/internal/services/users"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
	"uuid"

	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	log.Println("Starting cake...")

	if len(os.Getenv("JWT_SECRET")) == 0 {
		log.Fatalln("No JWT secret in $JWT_SECRET")
	}

	db, err := repository.NewDbConnection()
	if err != nil {
		log.Fatalf("error while opening connection to database: %s", fmt.Errorf("%w", err))
	}
	defer db.Close()

	firstTimeSetup(db)

	mux := http.NewServeMux()
	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	usersService := users.New(db)
	_ = userHandler.New(usersService, mux)

	providerEncryptor, err := encryptor.New()
	if err != nil {
		log.Fatalf("error while initializing provider encryption: %s", fmt.Errorf("%w", err))
	}
	providersService := providerService.New(db, providerEncryptor)
	_ = providerHandler.New(providersService, mux)

	log.Println("Cake server started and listening on port 8080")
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("error while http server was running: %s", fmt.Errorf("%w", err))
	}
}

// firstTimeSetup checks if the database is empty and creates an admin user
// if no users exist. The admin password is randomly generated and logged.
func firstTimeSetup(db *sqlx.DB) {
	var userCount int
	err := db.Get(&userCount, "SELECT COUNT(*) FROM users LIMIT 1")
	if err != nil {
		log.Fatalln(err)
	}

	if userCount != 0 {
		return
	}

	password, err := generatePassword()
	if err != nil {
		log.Fatalln(err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalln(err)
	}

	var roleID int
	err = db.Get(&roleID, `SELECT id FROM user_roles WHERE name = 'admin' LIMIT 1`)
	if err != nil {
		log.Fatalln(err)
	}

	user := userRepo.User{
		Uuid:         uuid.NewV4().String(),
		Username:     "admin",
		PasswordHash: hash,
		RoleId:       roleID,
		FirstName:    "admin",
		LastName:     "admin",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	_, err = db.NamedExec(
		`INSERT INTO users (
                   uuid,
                   username,
                   password_hash,
                   first_name,
                   last_name,
                   created_at,
                   updated_at,
                   role_id)
		VALUES (
		        :uuid,
		        :username,
		        :password_hash,
		        :first_name,
		        :last_name,
		        :created_at,
		        :updated_at,
		        :role_id)`,
		user)
	if err != nil {
		log.Fatalln(err)
	}

	log.Println("New admin user created\nusername: admin\npassword: " + password)
	log.Println("Change this password IMMEDIATELY")
}

// generatePassword creates a random 32-byte password encoded as base64.
func generatePassword() (string, error) {
	b := make([]byte, 24)

	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(b), nil
}
