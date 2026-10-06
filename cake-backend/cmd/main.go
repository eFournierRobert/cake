package main

import (
	"crypto/rand"
	"efournierrobert/cake-backend/internal/repository"
	userRepo "efournierrobert/cake-backend/internal/repository/users"
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

	log.Println("Cake server started and listening on port 8080")
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("error while http server was running: %s", fmt.Errorf("%w", err))
	}
}

func firstTimeSetup(db *sqlx.DB) {
	var userCount int
	err := db.Select(&userCount, "SELECT COUNT(*) FROM users")
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
	err = db.Get(&roleID, `SELECT id FROM user_roles WHERE name = 'admin'`)
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

func generatePassword() (string, error) {
	b := make([]byte, 24)

	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(b), nil
}
