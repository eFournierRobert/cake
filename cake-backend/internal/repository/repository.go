// Package repository is the database access layer: connection
// setup, goose migrations and a subpackage of repositories per
// domain.
package repository

import (
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/pressly/goose"
)

var databaseDriver = "mysql"

// NewDbConnection opens the database given by the DATABASE_URL
// environment variable and applies pending migrations from the
// migrations/ directory, which is resolved relative to the working
// directory.
func NewDbConnection() (*sqlx.DB, error) {
	db, err := sqlx.Connect(databaseDriver, os.Getenv("DATABASE_URL"))
	if err != nil {
		return nil, err
	}

	log.Println("opened connection to database successfully")

	if err := goose.SetDialect("mysql"); err != nil {
		return nil, err
	}

	if err := goose.Up(db.DB, "migrations"); err != nil {
		return nil, err
	}

	log.Println("Migrations ran successfully")

	return db, nil
}
